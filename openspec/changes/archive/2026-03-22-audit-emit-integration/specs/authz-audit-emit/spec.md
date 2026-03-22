## ADDED Requirements

### Requirement: Authz middleware supports audit recorder injection
`pkg/authz` SHALL provide a `WithAuditRecorder(r *audit.Recorder)` option that injects an audit recorder into the middleware configuration. When the recorder is nil, audit emission SHALL be skipped without error.

#### Scenario: Recorder injected via option
- **WHEN** `Authz(WithFGAClient(c), WithAuthzRules(rules), WithAuditRecorder(recorder))` is called
- **THEN** the middleware SHALL hold the recorder for use after each authorization check

#### Scenario: Nil recorder is safe
- **WHEN** `Authz(WithFGAClient(c), WithAuthzRules(rules), WithAuditRecorder(nil))` is called
- **THEN** the middleware SHALL function normally without emitting audit events

### Requirement: Authz middleware emits authz.decision event after Check
After each authorization Check (whether allowed, denied, or errored), the middleware SHALL call `recorder.RecordAuthzDecision` with the operation name, actor, and an `AuthzDetail` containing Relation, ObjectType, ObjectID, Decision, CacheHit, and ErrorReason.

#### Scenario: Allowed check emits event
- **WHEN** a request passes authorization (Check returns allowed=true)
- **THEN** the middleware SHALL emit an authz.decision event with `Decision: "allowed"` and `CacheHit` reflecting the actual cache status

#### Scenario: Denied check emits event
- **WHEN** a request fails authorization (Check returns allowed=false)
- **THEN** the middleware SHALL emit an authz.decision event with `Decision: "denied"` before returning the permission error

#### Scenario: Check error emits event
- **WHEN** the OpenFGA Check call returns an error
- **THEN** the middleware SHALL emit an authz.decision event with `Decision: "error"` and `ErrorReason` containing the error message

#### Scenario: No recorder skips emission
- **WHEN** the middleware has no recorder configured (nil)
- **AND** a request is processed
- **THEN** authorization SHALL proceed normally without any audit emission

### Requirement: Authz middleware passes full principal to openfga Check
The middleware SHALL construct the full OpenFGA principal string (e.g. `"user:" + actor.ID()`) before passing it to `openfga.CachedCheck`, instead of passing a bare user ID.

#### Scenario: User actor principal construction
- **WHEN** a request from a user actor with ID `"abc-123"` is authorized
- **THEN** the middleware SHALL call CachedCheck with user parameter `"user:abc-123"`
