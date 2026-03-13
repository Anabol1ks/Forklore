package grpcserver

import (
	"context"

	"github.com/Anabol1ks/Forklore/pkg/utils/authjwt"
)

type claimsContextKey struct{}

func WithClaims(ctx context.Context, claims *authjwt.AccessClaims) context.Context {
	return context.WithValue(ctx, claimsContextKey{}, claims)
}

func ClaimsFromContext(ctx context.Context) (*authjwt.AccessClaims, bool) {
	claims, ok := ctx.Value(claimsContextKey{}).(*authjwt.AccessClaims)
	return claims, ok
}
