package grpcserver

import (
	"auth-service/internal/domain"
	"auth-service/internal/security"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func ToGRPCError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, domain.ErrInvalidCredentials):
		return status.Error(codes.Unauthenticated, err.Error())

	case errors.Is(err, domain.ErrUnauthorized):
		return status.Error(codes.Unauthenticated, err.Error())

	case errors.Is(err, domain.ErrUserBlocked):
		return status.Error(codes.PermissionDenied, err.Error())

	case errors.Is(err, domain.ErrUsernameTaken),
		errors.Is(err, domain.ErrEmailTaken),
		errors.Is(err, domain.ErrUserAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())

	case errors.Is(err, domain.ErrUserNotFound),
		errors.Is(err, domain.ErrSessionNotFound),
		errors.Is(err, gorm.ErrRecordNotFound):
		return status.Error(codes.NotFound, err.Error())

	case errors.Is(err, domain.ErrSessionExpired),
		errors.Is(err, domain.ErrSessionRevoked),
		errors.Is(err, security.ErrInvalidAccessToken):
		return status.Error(codes.Unauthenticated, err.Error())

	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
