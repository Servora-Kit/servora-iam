package actor

// Scope key constants: conventional request-scope dimensions used by gateway and IAM.
// They are not the full domain model — only the keys used to pass scope from headers.
const (
	ScopeKeyTenantID       = "tenant_id"
	ScopeKeyOrganizationID = "organization_id"
	ScopeKeyProjectID      = "project_id"
)

// UserActor is the concrete actor for an authenticated user.
// Scope is stored as key-value; TenantID/OrganizationID/ProjectID are convenience accessors.
type UserActor struct {
	id          string
	displayName string
	email       string
	metadata    map[string]string
	scope       map[string]string
}

func NewUserActor(id, displayName, email string, metadata map[string]string) *UserActor {
	return &UserActor{
		id:          id,
		displayName: displayName,
		email:       email,
		metadata:    metadata,
		scope:       make(map[string]string),
	}
}

func (u *UserActor) ID() string           { return u.id }
func (u *UserActor) Type() Type          { return TypeUser }
func (u *UserActor) DisplayName() string { return u.displayName }
func (u *UserActor) Email() string      { return u.email }

func (u *UserActor) Scope(key string) string {
	if u.scope == nil {
		return ""
	}
	return u.scope[key]
}

func (u *UserActor) SetScope(key, value string) {
	if u.scope == nil {
		u.scope = make(map[string]string)
	}
	u.scope[key] = value
}

func (u *UserActor) Metadata() map[string]string {
	if u.metadata == nil {
		return map[string]string{}
	}
	return u.metadata
}

func (u *UserActor) Meta(key string) string {
	if u.metadata == nil {
		return ""
	}
	return u.metadata[key]
}

func (u *UserActor) TenantID() string       { return u.Scope(ScopeKeyTenantID) }
func (u *UserActor) OrganizationID() string { return u.Scope(ScopeKeyOrganizationID) }
func (u *UserActor) ProjectID() string      { return u.Scope(ScopeKeyProjectID) }
func (u *UserActor) SetTenantID(id string)       { u.SetScope(ScopeKeyTenantID, id) }
func (u *UserActor) SetOrganizationID(id string) { u.SetScope(ScopeKeyOrganizationID, id) }
func (u *UserActor) SetProjectID(id string)     { u.SetScope(ScopeKeyProjectID, id) }
