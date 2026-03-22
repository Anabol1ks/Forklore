package grpcserver

import (
	"errors"
	"profile-service/internal/domain"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	"gorm.io/gorm"

	"github.com/Anabol1ks/Forklore/pkg/utils/authjwt"
)

func ToGRPCError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, domain.ErrUnauthorized):
		return grpcstatus.Error(codes.Unauthenticated, err.Error())

	case errors.Is(err, domain.ErrProfileAccessDenied),
		errors.Is(err, domain.ErrSocialLinkAccessDenied):
		return grpcstatus.Error(codes.PermissionDenied, err.Error())

	case errors.Is(err, domain.ErrProfileNotFound),
		errors.Is(err, domain.ErrSocialLinkNotFound),
		errors.Is(err, domain.ErrProfileTitleNotFound),
		errors.Is(err, gorm.ErrRecordNotFound):
		return grpcstatus.Error(codes.NotFound, err.Error())

	case errors.Is(err, domain.ErrInvalidDisplayName),
		errors.Is(err, domain.ErrInvalidUsername),
		errors.Is(err, domain.ErrProfileTitleInactive),
		errors.Is(err, domain.ErrInvalidSocialPlatform),
		errors.Is(err, domain.ErrInvalidSocialURL),
		errors.Is(err, domain.ErrCannotFollowSelf):
		return grpcstatus.Error(codes.InvalidArgument, err.Error())

	case errors.Is(err, authjwt.ErrInvalidAccessToken):
		return grpcstatus.Error(codes.Unauthenticated, err.Error())

	default:
		return grpcstatus.Error(codes.Internal, "internal server error")
	}
}

func LogAndMapError(logger *zap.Logger, operation string, err error, fields ...zap.Field) error {
	if logger == nil {
		logger = zap.NewNop()
	}

	mapped := ToGRPCError(err)
	code := grpcstatus.Code(mapped)

	logFields := append(fields,
		zap.Error(err),
		zap.String("grpc_code", code.String()),
	)

	switch code {
	case codes.Internal:
		logger.Error(operation, logFields...)
	default:
		logger.Warn(operation, logFields...)
	}

	return mapped
}
