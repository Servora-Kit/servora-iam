package actor

// Type identifies the kind of request initiator (generic identity, not domain model).
type Type string

const (
	TypeUser      Type = "user"
	TypeSystem    Type = "system"
	TypeAnonymous Type = "anonymous"
)

// Actor represents the identity of a request initiator.
// Scope is a generic key-value bag for request-scope dimensions (e.g. tenant/org/project IDs
// from gateway headers); keys are platform convention, not full domain model.
type Actor interface {
	ID() string
	Type() Type
	DisplayName() string
	Scope(key string) string
}
