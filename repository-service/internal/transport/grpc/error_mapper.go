package grpcserver

import (
	"errors"
	"repository-service/internal/domain"

	"github.com/Anabol1ks/Forklore/pkg/utils/authjwt"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func ToGRPCError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, domain.ErrUnauthorized):
		return grpcstatus.Error(codes.Unauthenticated, err.Error())

	case errors.Is(err, domain.ErrRepositoryAccessDenied):
		return grpcstatus.Error(codes.PermissionDenied, err.Error())

	case errors.Is(err, domain.ErrRepositoryNotFound),
		errors.Is(err, domain.ErrTagNotFound),
		errors.Is(err, gorm.ErrRecordNotFound):
		return grpcstatus.Error(codes.NotFound, err.Error())

	case errors.Is(err, domain.ErrRepositorySlugTaken):
		return grpcstatus.Error(codes.AlreadyExists, err.Error())

	case errors.Is(err, domain.ErrInvalidRepositoryVisibility),
		errors.Is(err, domain.ErrInvalidRepositoryType),
		errors.Is(err, domain.ErrTagInactive):
		return grpcstatus.Error(codes.InvalidArgument, err.Error())

	case errors.Is(err, domain.ErrRepositoryCannotBeForked):
		return grpcstatus.Error(codes.FailedPrecondition, err.Error())

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
