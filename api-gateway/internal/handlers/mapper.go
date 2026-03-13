package handlers

import (
	"api-gateway/internal/models"
	"time"

	authv1 "github.com/Anabol1ks/Forklore/pkg/pb/auth/v1"
	commonv1 "github.com/Anabol1ks/Forklore/pkg/pb/common/v1"
	repositoryv1 "github.com/Anabol1ks/Forklore/pkg/pb/repository/v1"
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

// Repository mapping functions

func mapRepository(r *repositoryv1.Repository) models.RepositoryResponse {
	if r == nil {
		return models.RepositoryResponse{}
	}

	var createdAt, updatedAt, deletedAt string
	if r.GetCreatedAt() != nil {
		createdAt = r.GetCreatedAt().AsTime().Format(time.RFC3339)
	}
	if r.GetUpdatedAt() != nil {
		updatedAt = r.GetUpdatedAt().AsTime().Format(time.RFC3339)
	}

	var deletedAtPtr *string
	if r.GetDeletedAt() != nil {
		deletedAt = r.GetDeletedAt().AsTime().Format(time.RFC3339)
		deletedAtPtr = &deletedAt
	}

	var parentRepoID *string
	if r.GetParentRepoId() != nil && r.GetParentRepoId().GetValue() != "" {
		parentRepoID = &r.ParentRepoId.Value
	}

	resp := models.RepositoryResponse{
		RepoID:       r.GetRepoId().GetValue(),
		OwnerID:      r.GetOwnerId().GetValue(),
		Name:         r.GetName(),
		Slug:         r.GetSlug(),
		Description:  toPointerString(r.GetDescription()),
		Visibility:   mapRepositoryVisibility(r.GetVisibility()),
		Type:         mapRepositoryType(r.GetType()),
		Tag:          mapRepositoryTag(r.GetTag()),
		ParentRepoID: parentRepoID,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
		DeletedAt:    deletedAtPtr,
	}

	return resp
}

func mapRepositoryTag(t *repositoryv1.RepositoryTag) models.RepositoryTagResponse {
	if t == nil {
		return models.RepositoryTagResponse{}
	}

	var createdAt, updatedAt string
	if t.GetCreatedAt() != nil {
		createdAt = t.GetCreatedAt().AsTime().Format(time.RFC3339)
	}
	if t.GetUpdatedAt() != nil {
		updatedAt = t.GetUpdatedAt().AsTime().Format(time.RFC3339)
	}

	return models.RepositoryTagResponse{
		TagID:       t.GetTagId().GetValue(),
		Name:        t.GetName(),
		Slug:        t.GetSlug(),
		Description: t.GetDescription(),
		IsActive:    t.GetIsActive(),
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}

func mapRepositoryVisibility(v commonv1.RepositoryVisibility) string {
	switch v {
	case commonv1.RepositoryVisibility_REPOSITORY_VISIBILITY_PRIVATE:
		return "private"
	default:
		return "public"
	}
}

func mapRepositoryType(t commonv1.RepositoryType) string {
	switch t {
	case commonv1.RepositoryType_REPOSITORY_TYPE_NOTES:
		return "notes"
	case commonv1.RepositoryType_REPOSITORY_TYPE_MIXED:
		return "mixed"
	default:
		return "article"
	}
}

// toPointerString converts empty string to nil pointer.
func toPointerString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
