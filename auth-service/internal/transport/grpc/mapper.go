package grpcserver

import (
	model "auth-service/internal/models"
	"auth-service/internal/service"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	authv1 "github.com/Anabol1ks/Forklore/pkg/pb/auth/v1"
	commonv1 "github.com/Anabol1ks/Forklore/pkg/pb/common/v1"
)

func toProtoUUID(id uuid.UUID) *commonv1.UUID {
	return &commonv1.UUID{
		Value: id.String(),
	}
}

func toProtoTimestampPtr(t *time.Time) *timestamppb.Timestamp {
	if t == nil || t.IsZero() {
		return nil
	}
	return timestamppb.New(*t)
}

func toProtoTimestamp(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

func toProtoUserRole(role model.UserRole) commonv1.UserRole {
	switch role {
	case model.UserRoleAdmin:
		return commonv1.UserRole_USER_ROLE_ADMIN
	case model.UserRoleUser:
		fallthrough
	default:
		return commonv1.UserRole_USER_ROLE_USER
	}
}

func toProtoUserStatus(status model.UserStatus) commonv1.UserStatus {
	switch status {
	case model.UserStatusBlocked:
		return commonv1.UserStatus_USER_STATUS_BLOCKED
	case model.UserStatusDeleted:
		return commonv1.UserStatus_USER_STATUS_DELETED
	case model.UserStatusActive:
		fallthrough
	default:
		return commonv1.UserStatus_USER_STATUS_ACTIVE
	}
}

func toProtoUser(user *model.User) *authv1.User {
	if user == nil {
		return nil
	}

	return &authv1.User{
		UserId:      toProtoUUID(user.ID),
		Username:    user.Username,
		Email:       user.Email,
		Role:        toProtoUserRole(user.Role),
		Status:      toProtoUserStatus(user.Status),
		CreatedAt:   toProtoTimestamp(user.CreatedAt),
		UpdatedAt:   toProtoTimestamp(user.UpdatedAt),
		LastLoginAt: toProtoTimestampPtr(user.LastLoginAt),
	}
}

func toProtoTokenPair(tokens service.TokenPair) *authv1.TokenPair {
	return &authv1.TokenPair{
		AccessToken:      tokens.AccessToken,
		RefreshToken:     tokens.RefreshToken,
		TokenType:        tokens.TokenType,
		AccessExpiresAt:  toProtoTimestamp(tokens.AccessExpiresAt),
		RefreshExpiresAt: toProtoTimestamp(tokens.RefreshExpiresAt),
		SessionId:        toProtoUUID(tokens.SessionID),
	}
}

func toProtoAuthResponse(out *service.AuthOutput) *authv1.AuthResponse {
	if out == nil {
		return nil
	}

	return &authv1.AuthResponse{
		User:   toProtoUser(out.User),
		Tokens: toProtoTokenPair(out.Tokens),
	}
}

func toProtoIntrospectResponse(out *service.IntrospectOutput) *authv1.IntrospectResponse {
	if out == nil {
		return &authv1.IntrospectResponse{
			Active: false,
		}
	}

	return &authv1.IntrospectResponse{
		Active:    out.Active,
		UserId:    toProtoUUID(out.UserID),
		SessionId: toProtoUUID(out.SessionID),
		Username:  out.Username,
		Email:     out.Email,
		Role:      toProtoUserRole(out.Role),
		Status:    toProtoUserStatus(out.Status),
		ExpiresAt: toProtoTimestamp(out.ExpiresAt),
	}
}
