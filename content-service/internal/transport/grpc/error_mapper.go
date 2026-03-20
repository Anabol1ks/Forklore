package grpcserver

import (
	"content-service/internal/domain"
	"errors"

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

	case errors.Is(err, domain.ErrRepositoryNotFound):
		return grpcstatus.Error(codes.NotFound, err.Error())

	case errors.Is(err, domain.ErrContentAccessDenied):
		return grpcstatus.Error(codes.PermissionDenied, err.Error())

	case errors.Is(err, domain.ErrDocumentNotFound),
		errors.Is(err, domain.ErrDocumentVersionNotFound),
		errors.Is(err, domain.ErrFileNotFound),
		errors.Is(err, domain.ErrFileVersionNotFound),
		errors.Is(err, gorm.ErrRecordNotFound):
		return grpcstatus.Error(codes.NotFound, err.Error())

	case errors.Is(err, domain.ErrDocumentSlugTaken):
		return grpcstatus.Error(codes.AlreadyExists, err.Error())

	case errors.Is(err, domain.ErrInvalidDocumentFormat),
		errors.Is(err, domain.ErrInvalidDocumentTitle),
		errors.Is(err, domain.ErrInvalidDocumentSlug),
		errors.Is(err, domain.ErrInvalidFileName):
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
