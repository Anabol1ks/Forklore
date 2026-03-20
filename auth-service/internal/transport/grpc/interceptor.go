package grpcserver

import (
	"auth-service/internal/security"
	"context"
	"strings"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// LoggingInterceptor логирует каждый входящий gRPC запрос.
type LoggingInterceptor struct {
	log *zap.Logger
}

func NewLoggingInterceptor(log *zap.Logger) *LoggingInterceptor {
	return &LoggingInterceptor{log: log}
}

func (i *LoggingInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		dur := time.Since(start)

		code := codes.OK
		if err != nil {
			if st, ok := status.FromError(err); ok {
				code = st.Code()
			} else {
				code = codes.Internal
			}
		}

		logFn := i.log.Info
		if code != codes.OK {
			logFn = i.log.Warn
		}
		logFn("gRPC",
			zap.String("method", info.FullMethod),
			zap.String("code", code.String()),
			zap.Duration("dur", dur),
		)

		return resp, err
	}
}

type AuthInterceptor struct {
	tokenManager     security.TokenManager
	protectedMethods map[string]struct{}
}

func NewAuthInterceptor(tokenManager security.TokenManager) *AuthInterceptor {
	return &AuthInterceptor{
		tokenManager: tokenManager,
		protectedMethods: map[string]struct{}{
			"/forklore.auth.v1.AuthService/GetMe":     {},
			"/forklore.auth.v1.AuthService/LogoutAll": {},
		},
	}
}

func (i *AuthInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if _, ok := i.protectedMethods[info.FullMethod]; !ok {
			return handler(ctx, req)
		}

		token, err := extractBearerTokenFromMetadata(ctx)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}

		claims, err := i.tokenManager.ParseAccessToken(token)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid access token")
		}

		ctx = WithClaims(ctx, claims)

		return handler(ctx, req)
	}
}

func extractBearerTokenFromMetadata(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return "", status.Error(codes.Unauthenticated, "missing authorization header")
	}

	raw := strings.TrimSpace(values[0])
	parts := strings.SplitN(raw, " ", 2)
	if len(parts) != 2 {
		return "", status.Error(codes.Unauthenticated, "invalid authorization header")
	}

	if !strings.EqualFold(parts[0], "Bearer") {
		return "", status.Error(codes.Unauthenticated, "invalid authorization scheme")
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", status.Error(codes.Unauthenticated, "empty bearer token")
	}

	return token, nil
}
