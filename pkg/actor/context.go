package actor

import "context"

type contextKey struct{}

func NewContext(ctx context.Context, a Actor) context.Context {
	return context.WithValue(ctx, contextKey{}, a)
}

func FromContext(ctx context.Context) (Actor, bool) {
	a, ok := ctx.Value(contextKey{}).(Actor)
	return a, ok
}

// MustFromContext panics if no actor in context — use only in trusted code paths.
func MustFromContext(ctx context.Context) Actor {
	a, ok := FromContext(ctx)
	if !ok {
		panic("actor: no actor in context")
	}
	return a
}

// TenantIDFromContext returns the tenant scope from the actor in context (uses Scope(ScopeKeyTenantID)).
func TenantIDFromContext(ctx context.Context) (string, bool) {
	a, ok := FromContext(ctx)
	if !ok {
		return "", false
	}
	id := a.Scope(ScopeKeyTenantID)
	if id == "" {
		return "", false
	}
	return id, true
}

// OrganizationIDFromContext returns the organization scope from the actor in context.
func OrganizationIDFromContext(ctx context.Context) (string, bool) {
	a, ok := FromContext(ctx)
	if !ok {
		return "", false
	}
	id := a.Scope(ScopeKeyOrganizationID)
	if id == "" {
		return "", false
	}
	return id, true
}

// ProjectIDFromContext returns the project scope from the actor in context.
func ProjectIDFromContext(ctx context.Context) (string, bool) {
	a, ok := FromContext(ctx)
	if !ok {
		return "", false
	}
	id := a.Scope(ScopeKeyProjectID)
	if id == "" {
		return "", false
	}
	return id, true
}
