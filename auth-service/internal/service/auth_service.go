package service

import (
	"auth-service/internal/domain"
	model "auth-service/internal/models"
	"auth-service/internal/repository"
	"auth-service/internal/security"
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService interface {
	Register(ctx context.Context, input RegisterInput) (*AuthOutput, error)
	Login(ctx context.Context, input LoginInput) (*AuthOutput, error)
	Refresh(ctx context.Context, input RefreshInput) (*AuthOutput, error)

	Logout(ctx context.Context, refreshToken string) error
	LogoutAll(ctx context.Context, userID uuid.UUID) error

	Introspect(ctx context.Context, accessToken string) (*IntrospectOutput, error)
	GetMe(ctx context.Context, userID uuid.UUID) (*model.User, error)
}

type authService struct {
	repos           *repository.Repository
	passwordManager security.PasswordManager
	tokenManager    security.TokenManager
	refreshTokenTTL time.Duration
}

func NewAuthService(
	repos *repository.Repository,
	passwordManager security.PasswordManager,
	tokenManager security.TokenManager,
	refreshTokenTTL time.Duration,
) AuthService {
	return &authService{
		repos:           repos,
		passwordManager: passwordManager,
		tokenManager:    tokenManager,
		refreshTokenTTL: refreshTokenTTL,
	}
}

func (s *authService) Register(ctx context.Context, input RegisterInput) (*AuthOutput, error) {
	email := normalizeEmail(input.Email)

	if _, err := s.repos.User.GetByUsername(ctx, input.Username); err == nil {
		return nil, domain.ErrUsernameTaken
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if _, err := s.repos.User.GetByEmail(ctx, email); err == nil {
		return nil, domain.ErrEmailTaken
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	passwordHash, err := s.passwordManager.Hash(input.Password)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	var result *AuthOutput

	err = s.repos.Transaction(ctx, func(txRepo *repository.Repository) error {
		user := &model.User{
			Username:     input.Username,
			Email:        email,
			PasswordHash: passwordHash,
			Role:         model.UserRoleUser,
			Status:       model.UserStatusActive,
			LastLoginAt:  &now,
		}

		if err := txRepo.User.Create(ctx, user); err != nil {
			if mapped := mapCreateUserError(err); mapped != nil {
				return mapped
			}
			return err
		}

		rawRefreshToken, refreshTokenHash, err := s.tokenManager.GenerateRefreshToken()
		if err != nil {
			return err
		}

		session := &model.RefreshSession{
			UserID:     user.ID,
			TokenHash:  refreshTokenHash,
			DeviceName: &input.Meta.DeviceName,
			UserAgent:  &input.Meta.UserAgent,
			IP:         &input.Meta.IP,
			ExpiresAt:  now.Add(s.refreshTokenTTL),
		}

		if err := txRepo.Session.Create(ctx, session); err != nil {
			return err
		}

		accessToken, accessExpiresAt, err := s.tokenManager.GenerateAccessToken(
			user.ID,
			session.ID,
			user.Username,
			user.Email,
			user.Role,
			user.Status,
		)
		if err != nil {
			return err
		}

		result = &AuthOutput{
			User: user,
			Tokens: TokenPair{
				AccessToken:      accessToken,
				RefreshToken:     rawRefreshToken,
				TokenType:        "Bearer",
				AccessExpiresAt:  accessExpiresAt,
				RefreshExpiresAt: session.ExpiresAt,
				SessionID:        session.ID,
			},
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *authService) Login(ctx context.Context, input LoginInput) (*AuthOutput, error) {
	login := input.Login
	if strings.Contains(input.Login, "@") {
		login = normalizeEmail(input.Login)
	}

	user, err := s.repos.User.GetByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, err
	}

	if err := s.passwordManager.Compare(user.PasswordHash, input.Password); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, err
	}

	if err := ensureUserCanAuthenticate(user); err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	var result *AuthOutput

	err = s.repos.Transaction(ctx, func(txRepo *repository.Repository) error {
		if err := txRepo.User.UpdateLastLoginAt(ctx, user.ID, now); err != nil {
			return err
		}
		user.LastLoginAt = &now

		rawRefreshToken, refreshTokenHash, err := s.tokenManager.GenerateRefreshToken()
		if err != nil {
			return err
		}

		session := &model.RefreshSession{
			UserID:     user.ID,
			TokenHash:  refreshTokenHash,
			DeviceName: stringPtr(input.Meta.DeviceName),
			UserAgent:  stringPtr(input.Meta.UserAgent),
			IP:         stringPtr(input.Meta.IP),
			ExpiresAt:  now.Add(s.refreshTokenTTL),
		}

		if err := txRepo.Session.Create(ctx, session); err != nil {
			return err
		}

		accessToken, accessExpiresAt, err := s.tokenManager.GenerateAccessToken(
			user.ID,
			session.ID,
			user.Username,
			user.Email,
			user.Role,
			user.Status,
		)
		if err != nil {
			return err
		}

		result = &AuthOutput{
			User: user,
			Tokens: TokenPair{
				AccessToken:      accessToken,
				RefreshToken:     rawRefreshToken,
				TokenType:        "Bearer",
				AccessExpiresAt:  accessExpiresAt,
				RefreshExpiresAt: session.ExpiresAt,
				SessionID:        session.ID,
			},
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *authService) Refresh(ctx context.Context, input RefreshInput) (*AuthOutput, error) {
	refreshTokenHash := s.tokenManager.HashRefreshToken(input.RefreshToken)

	now := time.Now().UTC()
	var result *AuthOutput

	err := s.repos.Transaction(ctx, func(txRepo *repository.Repository) error {
		currentSession, err := txRepo.Session.GetByTokenHash(ctx, refreshTokenHash)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrSessionNotFound
			}
			return err
		}

		if currentSession.RevokedAt != nil {
			return domain.ErrSessionRevoked
		}

		if now.After(currentSession.ExpiresAt) {
			return domain.ErrSessionExpired
		}

		user, err := txRepo.User.GetByID(ctx, currentSession.UserID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrUserNotFound
			}
			return err
		}

		if err := ensureUserCanAuthenticate(user); err != nil {
			return err
		}

		if err := txRepo.Session.RevokeByID(ctx, currentSession.ID, now); err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}

		rawRefreshToken, newRefreshTokenHash, err := s.tokenManager.GenerateRefreshToken()
		if err != nil {
			return err
		}

		rotatedFrom := currentSession.ID
		newSession := &model.RefreshSession{
			UserID:               user.ID,
			TokenHash:            newRefreshTokenHash,
			DeviceName:           coalesceStringPtr(input.Meta.DeviceName, currentSession.DeviceName),
			UserAgent:            coalesceStringPtr(input.Meta.UserAgent, currentSession.UserAgent),
			IP:                   coalesceStringPtr(input.Meta.IP, currentSession.IP),
			ExpiresAt:            now.Add(s.refreshTokenTTL),
			RotatedFromSessionID: &rotatedFrom,
		}

		if err := txRepo.Session.Create(ctx, newSession); err != nil {
			return err
		}

		accessToken, accessExpiresAt, err := s.tokenManager.GenerateAccessToken(
			user.ID,
			newSession.ID,
			user.Username,
			user.Email,
			user.Role,
			user.Status,
		)
		if err != nil {
			return err
		}

		result = &AuthOutput{
			User: user,
			Tokens: TokenPair{
				AccessToken:      accessToken,
				RefreshToken:     rawRefreshToken,
				TokenType:        "Bearer",
				AccessExpiresAt:  accessExpiresAt,
				RefreshExpiresAt: newSession.ExpiresAt,
				SessionID:        newSession.ID,
			},
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	refreshTokenHash := s.tokenManager.HashRefreshToken(refreshToken)

	session, err := s.repos.Session.GetByTokenHash(ctx, refreshTokenHash)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	if session.RevokedAt != nil {
		return nil
	}

	return s.repos.Session.RevokeByID(ctx, session.ID, time.Now().UTC())
}

func (s *authService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	if userID == uuid.Nil {
		return domain.ErrUnauthorized
	}

	return s.repos.Session.RevokeAllByUserID(ctx, userID, time.Now().UTC())
}

func (s *authService) Introspect(ctx context.Context, accessToken string) (*IntrospectOutput, error) {
	claims, err := s.tokenManager.ParseAccessToken(accessToken)
	if err != nil {
		return &IntrospectOutput{
			Active: false,
		}, nil
	}

	return &IntrospectOutput{
		Active:    true,
		UserID:    claims.UserID,
		SessionID: claims.SessionID,
		Username:  claims.Username,
		Email:     claims.Email,
		Role:      claims.Role,
		Status:    claims.Status,
		ExpiresAt: claims.ExpiresAt,
	}, nil
}

func (s *authService) GetMe(ctx context.Context, userID uuid.UUID) (*model.User, error) {
	if userID == uuid.Nil {
		return nil, domain.ErrUnauthorized
	}

	user, err := s.repos.User.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}

	return user, nil
}

func normalizeEmail(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func mapCreateUserError(err error) error {
	if err == nil {
		return nil
	}

	msg := strings.ToLower(err.Error())

	switch {
	case strings.Contains(msg, "users_username_key"),
		strings.Contains(msg, "username"):
		return domain.ErrUsernameTaken
	case strings.Contains(msg, "users_email_key"),
		strings.Contains(msg, "email"):
		return domain.ErrEmailTaken
	case strings.Contains(msg, "duplicate key"):
		return domain.ErrUserAlreadyExists
	default:
		return nil
	}
}

func ensureUserCanAuthenticate(user *model.User) error {
	switch user.Status {
	case model.UserStatusActive:
		return nil
	case model.UserStatusBlocked:
		return domain.ErrUserBlocked
	default:
		return domain.ErrUnauthorized
	}
}
func coalesceStringPtr(candidate string, fallback *string) *string {
	if v := stringPtr(candidate); v != nil {
		return v
	}
	return fallback
}
func stringPtr(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}
