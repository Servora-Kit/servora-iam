## ADDED Requirements

### Requirement: Actor interface provides identity-level fields

`Actor` interface SHALL expose the following methods beyond the existing `ID()`, `Type()`, `DisplayName()`:
- `Subject() string` — 外部 IdP subject identifier
- `ClientID() string` — OAuth2 client identifier
- `Realm() string` — IdP realm / tenant namespace
- `Email() string` — 邮箱地址
- `Roles() []string` — 角色列表
- `Scopes() []string` — OAuth2 scope 列表
- `Attrs() map[string]string` — 扩展属性 bag

现有的 `Scope(key string) string` 方法 SHALL 保留，用于请求级维度（tenant/org/project）。

#### Scenario: UserActor carries all identity fields

- **WHEN** a `UserActor` is constructed with id, displayName, email, subject, clientID, realm, roles, scopes, and attrs
- **THEN** all getter methods SHALL return the corresponding values

#### Scenario: Missing optional fields return zero values

- **WHEN** a `UserActor` is constructed with only id and type
- **THEN** `Email()` SHALL return `""`, `Roles()` SHALL return `nil` or empty slice, `Attrs()` SHALL return empty map

### Requirement: ServiceActor type for service-to-service calls

The system SHALL provide a `ServiceActor` implementing `Actor` with `Type()` returning `TypeService` ("service").

`ServiceActor` SHALL carry at minimum: `ID`, `ClientID`, `DisplayName`.

#### Scenario: ServiceActor is recognized by type

- **WHEN** a `ServiceActor` is constructed with id "order-svc" and clientID "order-client"
- **THEN** `Type()` SHALL return `TypeService`, `ID()` SHALL return "order-svc", `ClientID()` SHALL return "order-client"

### Requirement: Actor Type enum includes service type

The `Type` constants SHALL include `TypeService Type = "service"` in addition to existing `TypeUser`, `TypeSystem`, `TypeAnonymous`.

#### Scenario: Type constant availability

- **WHEN** code references `actor.TypeService`
- **THEN** it SHALL compile and equal the string "service"

### Requirement: Existing callers compile after migration

All existing code that creates `UserActor` or consumes `Actor` interface SHALL be updated to compile with the new interface signature. This includes `pkg/authn`, `pkg/transport/server/middleware`, `app/iam/service`, `app/sayhello/service`.

#### Scenario: pkg/authn defaultClaimsMapper adapts to new UserActor

- **WHEN** `defaultClaimsMapper` is called with JWT MapClaims containing "sub", "name", "email", "realm_access.roles"
- **THEN** it SHALL return a `UserActor` with `Subject()` populated from "sub" and `Roles()` populated from claims

#### Scenario: Full project compiles after actor v2

- **WHEN** `go build ./...` is run across all workspace modules
- **THEN** compilation SHALL succeed with zero errors
