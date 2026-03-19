package middleware

import (
	"testing"

	authzpb "github.com/Servora-Kit/servora/api/gen/go/authz/service/v1"
	iamv1 "github.com/Servora-Kit/servora/api/gen/go/iam/service/v1"
	userpb "github.com/Servora-Kit/servora/api/gen/go/user/service/v1"
)

func TestResolveObject_PlatformSingleton(t *testing.T) {
	rule := iamv1.AuthzRuleEntry{
		Mode:       authzpb.AuthzMode_AUTHZ_MODE_CHECK,
		ObjectType: "platform",
		Relation:   "admin",
		IDField:    "", // no IDField → singleton "default"
	}

	objType, objID, err := resolveObject(rule, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if objType != "platform" {
		t.Errorf("objectType = %q, want %q", objType, "platform")
	}
	if objID != "default" {
		t.Errorf("objectID = %q, want %q", objID, "default")
	}
}

func TestResolveObject_ObjectFromRequest(t *testing.T) {
	rule := iamv1.AuthzRuleEntry{
		Mode:       authzpb.AuthzMode_AUTHZ_MODE_CHECK,
		ObjectType: "user",
		Relation:   "admin",
		IDField:    "id",
	}
	req := &userpb.GetUserRequest{Id: "user-uuid-1"}

	objType, objID, err := resolveObject(rule, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if objType != "user" {
		t.Errorf("objectType = %q, want %q", objType, "user")
	}
	if objID != "user-uuid-1" {
		t.Errorf("objectID = %q, want %q", objID, "user-uuid-1")
	}
}

func TestResolveObject_MissingObjectType(t *testing.T) {
	rule := iamv1.AuthzRuleEntry{
		Mode:       authzpb.AuthzMode_AUTHZ_MODE_CHECK,
		ObjectType: "", // missing
		Relation:   "admin",
	}

	_, _, err := resolveObject(rule, nil)
	if err == nil {
		t.Fatal("expected error when ObjectType is empty")
	}
}

func TestResolveObject_IDFieldNotFoundInRequest(t *testing.T) {
	rule := iamv1.AuthzRuleEntry{
		Mode:       authzpb.AuthzMode_AUTHZ_MODE_CHECK,
		ObjectType: "user",
		Relation:   "admin",
		IDField:    "nonexistent_field",
	}
	req := &userpb.GetUserRequest{Id: "user-uuid-1"}

	_, _, err := resolveObject(rule, req)
	if err == nil {
		t.Fatal("expected error when IDField does not exist in request")
	}
}

func TestResolveObject_NonProtoRequest(t *testing.T) {
	rule := iamv1.AuthzRuleEntry{
		Mode:       authzpb.AuthzMode_AUTHZ_MODE_CHECK,
		ObjectType: "user",
		Relation:   "admin",
		IDField:    "id",
	}

	_, _, err := resolveObject(rule, "not-a-proto-message")
	if err == nil {
		t.Fatal("expected error for non-proto request")
	}
}
