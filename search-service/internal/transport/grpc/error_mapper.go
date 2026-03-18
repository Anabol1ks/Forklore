package grpcserver

import (
	"errors"
	"search-service/internal/domain"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

func ToGRPCError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, domain.ErrInvalidSearchQuery):
		return grpcstatus.Error(codes.InvalidArgument, err.Error())

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
