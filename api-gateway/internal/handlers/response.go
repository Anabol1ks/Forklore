package handlers

import (
	"net/http"

	"api-gateway/internal/models"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func handleGRPCError(err error) (int, models.ErrorResponse) {
	st, ok := status.FromError(err)
	if !ok {
		return http.StatusInternalServerError, models.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "internal error",
		}
	}

	httpCode := http.StatusInternalServerError
	switch st.Code() {
	case codes.InvalidArgument:
		httpCode = http.StatusBadRequest
	case codes.Unauthenticated:
		httpCode = http.StatusUnauthorized
	case codes.PermissionDenied:
		httpCode = http.StatusForbidden
	case codes.NotFound:
		httpCode = http.StatusNotFound
	case codes.AlreadyExists:
		httpCode = http.StatusConflict
	case codes.Unavailable:
		httpCode = http.StatusServiceUnavailable
	case codes.DeadlineExceeded:
		httpCode = http.StatusGatewayTimeout
	}

	msg := st.Message()
	if httpCode == http.StatusInternalServerError {
		msg = "internal error"
	}

	return httpCode, models.ErrorResponse{
		Code:    httpCode,
		Message: msg,
	}
}
