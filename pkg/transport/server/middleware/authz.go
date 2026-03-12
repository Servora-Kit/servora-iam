package middleware

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	authzpb "github.com/Servora-Kit/servora/api/gen/go/servora/authz/v1"
	"github.com/Servora-Kit/servora/pkg/actor"
	"github.com/Servora-Kit/servora/pkg/openfga"
)

// AuthzRuleEntry mirrors the generated type so the middleware can consume rules
// from any proto-generated service package.
type AuthzRuleEntry struct {
	Mode       authzpb.AuthzMode
	Relation   authzpb.Relation
	ObjectType authzpb.ObjectType
	IDField    string
}

// AuthzOption configures the Authz middleware.
type AuthzOption func(*authzConfig)

type authzConfig struct {
	fga        *openfga.Client
	rules      map[string]AuthzRuleEntry
	platRootID string
}

// WithFGAClient sets the OpenFGA client.
func WithFGAClient(c *openfga.Client) AuthzOption {
	return func(cfg *authzConfig) { cfg.fga = c }
}

// WithAuthzRules sets the operation->rule mapping (typically from generated code).
func WithAuthzRules(rules map[string]AuthzRuleEntry) AuthzOption {
	return func(cfg *authzConfig) { cfg.rules = rules }
}

// WithPlatformRootID sets the platform:root object ID for platform-level checks.
func WithPlatformRootID(id string) AuthzOption {
	return func(cfg *authzConfig) { cfg.platRootID = id }
}

// Authz creates a Kratos middleware that performs authorization checks
// using OpenFGA based on proto-declared rules.
//
// Behavior:
//   - AUTHZ_MODE_NONE: skip authorization
//   - AUTHZ_MODE_ORGANIZATION: check relation on organization:{id}
//   - AUTHZ_MODE_PROJECT: check relation on project:{id}
//   - AUTHZ_MODE_OBJECT: check relation on {object_type}:{id}
//   - No rule found (fail-closed): deny
//   - OpenFGA unavailable (fail-closed): 503
func Authz(opts ...AuthzOption) middleware.Middleware {
	cfg := &authzConfig{}
	for _, o := range opts {
		o(cfg)
	}

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return handler(ctx, req)
			}

			operation := tr.Operation()
			rule, found := cfg.rules[operation]
			if !found {
				return nil, errors.Forbidden("AUTHZ_NO_RULE",
					fmt.Sprintf("no authorization rule for operation %s", operation))
			}

			if rule.Mode == authzpb.AuthzMode_AUTHZ_MODE_NONE {
				return handler(ctx, req)
			}

			a, ok := actor.FromContext(ctx)
			if !ok || a.Type() != actor.TypeUser {
				return nil, errors.Forbidden("AUTHZ_DENIED", "authentication required")
			}
			userID := a.ID()

			if cfg.fga == nil {
				return nil, errors.ServiceUnavailable("AUTHZ_UNAVAILABLE", "authorization service not available")
			}

			objectType, objectID, err := resolveObject(rule, cfg.platRootID, req)
			if err != nil {
				return nil, errors.BadRequest("AUTHZ_BAD_REQUEST",
					fmt.Sprintf("cannot resolve authorization target: %v", err))
			}

			relation := relationToFGA(rule.Relation)
			allowed, err := cfg.fga.Check(ctx, userID, relation, objectType, objectID)
			if err != nil {
				return nil, errors.ServiceUnavailable("AUTHZ_CHECK_FAILED",
					fmt.Sprintf("authorization check failed: %v", err))
			}
			if !allowed {
				return nil, errors.Forbidden("AUTHZ_DENIED", "insufficient permissions")
			}

			return handler(ctx, req)
		}
	}
}

func resolveObject(rule AuthzRuleEntry, platRootID string, req any) (objectType, objectID string, err error) {
	switch rule.Mode {
	case authzpb.AuthzMode_AUTHZ_MODE_ORGANIZATION:
		objectType = "organization"
		objectID, err = extractProtoField(req, rule.IDField)
	case authzpb.AuthzMode_AUTHZ_MODE_PROJECT:
		objectType = "project"
		objectID, err = extractProtoField(req, rule.IDField)
	case authzpb.AuthzMode_AUTHZ_MODE_OBJECT:
		objectType = objectTypeToFGA(rule.ObjectType)
		if rule.IDField == "root" && objectType == "platform" {
			objectID = platRootID
		} else {
			objectID, err = extractProtoField(req, rule.IDField)
		}
	default:
		err = fmt.Errorf("unsupported authz mode: %v", rule.Mode)
	}
	return
}

// extractProtoField uses proto reflection to read a string field from the request message.
func extractProtoField(req any, fieldName string) (string, error) {
	if fieldName == "" {
		return "", fmt.Errorf("id_field not specified")
	}

	msg, ok := req.(proto.Message)
	if !ok {
		return "", fmt.Errorf("request is not a proto message")
	}

	md := msg.ProtoReflect().Descriptor()
	fd := md.Fields().ByName(protoreflect.Name(fieldName))
	if fd == nil {
		return "", fmt.Errorf("field %q not found in %s", fieldName, md.FullName())
	}

	val := msg.ProtoReflect().Get(fd)
	s := val.String()
	if s == "" {
		return "", fmt.Errorf("field %q is empty", fieldName)
	}
	return s, nil
}

func relationToFGA(r authzpb.Relation) string {
	s := strings.TrimPrefix(r.String(), "RELATION_")
	return strings.ToLower(s)
}

func objectTypeToFGA(ot authzpb.ObjectType) string {
	s := strings.TrimPrefix(ot.String(), "OBJECT_TYPE_")
	return strings.ToLower(s)
}
