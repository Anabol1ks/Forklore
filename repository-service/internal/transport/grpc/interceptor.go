package grpcserver

import (
	"context"
	"strings"

	"github.com/Anabol1ks/Forklore/pkg/utils/authjwt"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	grpcstatus "google.golang.org/grpc/status"
)

type AuthInterceptor struct {
	tokenManager     authjwt.TokenVerifier
	logger           *zap.Logger
	protectedMethods map[string]struct{}
}

func NewAuthInterceptor(tokenManager authjwt.TokenVerifier, logger *zap.Logger) *AuthInterceptor {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &AuthInterceptor{
		tokenManager: tokenManager,
		logger:       logger,
		protectedMethods: map[string]struct{}{
			"/forklore.repository.v1.RepositoryService/CreateRepository":   {},
			"/forklore.repository.v1.RepositoryService/UpdateRepository":   {},
			"/forklore.repository.v1.RepositoryService/DeleteRepository":   {},
			"/forklore.repository.v1.RepositoryService/ForkRepository":     {},
			"/forklore.repository.v1.RepositoryService/ListMyRepositories": {},
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
		_, protected := i.protectedMethods[info.FullMethod]

		token, present, err := extractBearerTokenFromMetadata(ctx)
		if err != nil {
			i.logger.Warn("failed to extract bearer token",
				zap.String("method", info.FullMethod),
				zap.Error(err),
			)
			return nil, grpcstatus.Error(codes.Unauthenticated, err.Error())
		}

		if !present {
			if protected {
				i.logger.Warn("missing authorization for protected method",
					zap.String("method", info.FullMethod),
				)
				return nil, grpcstatus.Error(codes.Unauthenticated, "missing authorization header")
			}

			return handler(ctx, req)
		}

		claims, err := i.tokenManager.ParseAccessToken(token)
		if err != nil {
			i.logger.Warn("failed to parse access token",
				zap.String("method", info.FullMethod),
				zap.Error(err),
			)
			return nil, grpcstatus.Error(codes.Unauthenticated, "invalid access token")
		}

		ctx = WithClaims(ctx, claims)
		return handler(ctx, req)
	}
}

func extractBearerTokenFromMetadata(ctx context.Context) (token string, present bool, err error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", false, nil
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return "", false, nil
	}

	raw := strings.TrimSpace(values[0])
	if raw == "" {
		return "", true, grpcstatus.Error(codes.Unauthenticated, "empty authorization header")
	}

	parts := strings.SplitN(raw, " ", 2)
	if len(parts) != 2 {
		return "", true, grpcstatus.Error(codes.Unauthenticated, "invalid authorization header")
	}

	if !strings.EqualFold(parts[0], "Bearer") {
		return "", true, grpcstatus.Error(codes.Unauthenticated, "invalid authorization scheme")
	}

	token = strings.TrimSpace(parts[1])
	if token == "" {
		return "", true, grpcstatus.Error(codes.Unauthenticated, "empty bearer token")
	}

	return token, true, nil
}
