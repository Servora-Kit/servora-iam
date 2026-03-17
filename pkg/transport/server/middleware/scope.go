package middleware

import (
	"context"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/google/uuid"

	"github.com/Servora-Kit/servora/pkg/actor"
)

const (
	TenantIDHeader       = "X-Tenant-ID"
	OrganizationIDHeader = "X-Organization-ID"
	ProjectIDHeader      = "X-Project-ID"
)

// ScopeFromHeaders creates a Kratos middleware that reads organization and
// project scope from request headers and injects them into the UserActor.
//
// Requires an authenticated UserActor in context (i.e. must run after Authn).
// Headers are optional — absent headers are silently skipped.
// Invalid UUID values result in a 400 error.
func ScopeFromHeaders() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return handler(ctx, req)
			}

			a, ok := actor.FromContext(ctx)
			if !ok {
				return handler(ctx, req)
			}
			ua, ok := a.(*actor.UserActor)
			if !ok {
				return handler(ctx, req)
			}

			if tenantID := tr.RequestHeader().Get(TenantIDHeader); tenantID != "" {
				if _, err := uuid.Parse(tenantID); err != nil {
					return nil, errors.BadRequest("INVALID_TENANT_ID",
						"invalid X-Tenant-ID header")
				}
				ua.SetTenantID(tenantID)
			}
			if orgID := tr.RequestHeader().Get(OrganizationIDHeader); orgID != "" {
				if _, err := uuid.Parse(orgID); err != nil {
					return nil, errors.BadRequest("INVALID_ORGANIZATION_ID",
						"invalid X-Organization-ID header")
				}
				ua.SetOrganizationID(orgID)
			}
			if projID := tr.RequestHeader().Get(ProjectIDHeader); projID != "" {
				if _, err := uuid.Parse(projID); err != nil {
					return nil, errors.BadRequest("INVALID_PROJECT_ID",
						"invalid X-Project-ID header")
				}
				ua.SetProjectID(projID)
			}

			return handler(ctx, req)
		}
	}
}
