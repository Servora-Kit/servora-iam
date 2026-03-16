package middleware

import (
	"context"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"

	"github.com/Servora-Kit/servora/pkg/actor"
)

// DefaultUserIDHeader is the header name used by the gateway (e.g. Traefik
// ForwardAuth) to propagate the authenticated user ID to upstream services.
const DefaultUserIDHeader = "X-User-ID"

// IdentityOption configures the IdentityFromHeader middleware.
type IdentityOption func(*identityConfig)

type identityConfig struct {
	headerKey string
}

// WithHeaderKey overrides the default header name ("X-User-ID").
func WithHeaderKey(key string) IdentityOption {
	return func(c *identityConfig) { c.headerKey = key }
}

// IdentityFromHeader creates a Kratos middleware that reads the user identity
// from a gateway-injected HTTP header and injects an actor.Actor into the
// request context.
//
// This is the lightweight counterpart of a full JWT Authn middleware: it trusts
// that the gateway has already performed token verification via ForwardAuth and
// simply propagates the resulting user ID.
//
// If the header is present and non-empty, a UserActor is injected; otherwise an
// AnonymousActor is injected.
func IdentityFromHeader(opts ...IdentityOption) middleware.Middleware {
	cfg := &identityConfig{headerKey: DefaultUserIDHeader}
	for _, o := range opts {
		o(cfg)
	}

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				ctx = actor.NewContext(ctx, actor.NewAnonymousActor())
				return handler(ctx, req)
			}

			userID := tr.RequestHeader().Get(cfg.headerKey)
			if userID == "" {
				ctx = actor.NewContext(ctx, actor.NewAnonymousActor())
				return handler(ctx, req)
			}

			ctx = actor.NewContext(ctx, actor.NewUserActor(userID, "", "", nil))
			return handler(ctx, req)
		}
	}
}
