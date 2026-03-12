package handlers

import (
	"api-gateway/internal/models"
	"time"

	authv1 "github.com/Anabol1ks/Forklore/pkg/pb/auth/v1"
	commonv1 "github.com/Anabol1ks/Forklore/pkg/pb/common/v1"
)

func mapAuthResponse(resp *authv1.AuthResponse) models.AuthResponse {
	return models.AuthResponse{
		User:   mapUser(resp.GetUser()),
		Tokens: mapTokenPair(resp.GetTokens()),
	}
}

func mapUser(u *authv1.User) models.UserResponse {
	if u == nil {
		return models.UserResponse{}
	}

	var lastLogin *string
	if u.GetLastLoginAt() != nil {
		s := u.GetLastLoginAt().AsTime().Format(time.RFC3339)
		lastLogin = &s
	}

	var updatedAt string
	if u.GetUpdatedAt() != nil {
		updatedAt = u.GetUpdatedAt().AsTime().Format(time.RFC3339)
	}

	var createdAt string
	if u.GetCreatedAt() != nil {
		createdAt = u.GetCreatedAt().AsTime().Format(time.RFC3339)
	}

	return models.UserResponse{
		UserID:      u.GetUserId().GetValue(),
		Username:    u.GetUsername(),
		Email:       u.GetEmail(),
		Role:        mapUserRole(u.GetRole()),
		Status:      mapUserStatus(u.GetStatus()),
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		LastLoginAt: lastLogin,
	}
}

func mapTokenPair(t *authv1.TokenPair) models.TokenPairResponse {
	if t == nil {
		return models.TokenPairResponse{}
	}

	var accessExp, refreshExp string
	if t.GetAccessExpiresAt() != nil {
		accessExp = t.GetAccessExpiresAt().AsTime().Format(time.RFC3339)
	}
	if t.GetRefreshExpiresAt() != nil {
		refreshExp = t.GetRefreshExpiresAt().AsTime().Format(time.RFC3339)
	}

	return models.TokenPairResponse{
		AccessToken:      t.GetAccessToken(),
		RefreshToken:     t.GetRefreshToken(),
		TokenType:        t.GetTokenType(),
		AccessExpiresAt:  accessExp,
		RefreshExpiresAt: refreshExp,
		SessionID:        t.GetSessionId().GetValue(),
	}
}

func mapUserRole(role commonv1.UserRole) string {
	switch role {
	case commonv1.UserRole_USER_ROLE_ADMIN:
		return "admin"
	default:
		return "user"
	}
}

func mapUserStatus(status commonv1.UserStatus) string {
	switch status {
	case commonv1.UserStatus_USER_STATUS_BLOCKED:
		return "blocked"
	case commonv1.UserStatus_USER_STATUS_DELETED:
		return "deleted"
	default:
		return "active"
	}
}
