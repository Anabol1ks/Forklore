package security

import (
	model "auth-service/internal/models"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidAccessToken = errors.New("invalid access token")
)

type TokenManager interface {
	GenerateAccessToken(
		userID uuid.UUID,
		sessionID uuid.UUID,
		username string,
		email string,
		role model.UserRole,
		status model.UserStatus,
	) (string, time.Time, error)

	ParseAccessToken(token string) (*AccessClaims, error)

	GenerateRefreshToken() (raw string, hash string, err error)
	HashRefreshToken(raw string) string
}

type JWTTokenManager struct {
	secret           []byte
	issuer           string
	accessTokenTTL   time.Duration
	refreshTokenSize int
}

type AccessClaims struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
	Username  string
	Email     string
	Role      model.UserRole
	Status    model.UserStatus
	ExpiresAt time.Time
}

type jwtClaims struct {
	Username  string           `json:"username"`
	Email     string           `json:"email"`
	Role      model.UserRole   `json:"role"`
	Status    model.UserStatus `json:"status"`
	SessionID string           `json:"sid"`
	jwt.RegisteredClaims
}

func NewJWTTokenManager(secret string, issuer string, accessTokenTTL time.Duration) *JWTTokenManager {
	if issuer == "" {
		issuer = "forklore-auth-service"
	}

	return &JWTTokenManager{
		secret:           []byte(secret),
		issuer:           issuer,
		accessTokenTTL:   accessTokenTTL,
		refreshTokenSize: 32,
	}
}

func (m *JWTTokenManager) GenerateAccessToken(
	userID uuid.UUID,
	sessionID uuid.UUID,
	username string,
	email string,
	role model.UserRole,
	status model.UserStatus,
) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(m.accessTokenTTL)

	claims := jwtClaims{
		Username:  username,
		Email:     email,
		Role:      role,
		Status:    status,
		SessionID: sessionID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, err
	}

	return signed, expiresAt, nil
}

func (m *JWTTokenManager) ParseAccessToken(tokenString string) (*AccessClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidAccessToken
		}

		return m.secret, nil
	})
	if err != nil {
		return nil, ErrInvalidAccessToken
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidAccessToken
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, ErrInvalidAccessToken
	}

	sessionID, err := uuid.Parse(claims.SessionID)
	if err != nil {
		return nil, ErrInvalidAccessToken
	}

	if claims.ExpiresAt == nil {
		return nil, ErrInvalidAccessToken
	}

	return &AccessClaims{
		UserID:    userID,
		SessionID: sessionID,
		Username:  claims.Username,
		Email:     claims.Email,
		Role:      claims.Role,
		Status:    claims.Status,
		ExpiresAt: claims.ExpiresAt.Time,
	}, nil
}

func (m *JWTTokenManager) GenerateRefreshToken() (raw string, hash string, err error) {
	b := make([]byte, m.refreshTokenSize)

	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}

	raw = base64.RawURLEncoding.EncodeToString(b)
	hash = m.HashRefreshToken(raw)

	return raw, hash, nil
}

func (m *JWTTokenManager) HashRefreshToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
