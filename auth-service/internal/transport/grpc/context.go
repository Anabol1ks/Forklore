package grpcserver

import (
	"auth-service/internal/security"
	"context"
)

type claimsContextKey struct{}

func WithClaims(ctx context.Context, claims *security.AccessClaims) context.Context {
	return context.WithValue(ctx, claimsContextKey{}, claims)
}

func ClaimsFromContext(ctx context.Context) (*security.AccessClaims, bool) {
	claims, ok := ctx.Value(claimsContextKey{}).(*security.AccessClaims)
	return claims, ok
}
