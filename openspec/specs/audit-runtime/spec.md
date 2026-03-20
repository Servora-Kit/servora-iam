# Spec: audit-runtime

## Purpose

Defines requirements for the `audit-runtime` capability.

## Requirements

### Requirement: AuditEvent model covers four event types

`pkg/audit` SHALL define an `AuditEvent` struct with the following stable skeleton fields:
- `EventID` (string, UUID)
- `EventType` (enum: `AuthnResult`, `AuthzDecision`, `TupleChanged`, `ResourceMutation`)
- `Version` (string)
- `OccurredAt` (time.Time)
- `Service` (string)
- `Operation` (string)
- `Actor` (ActorInfo snapshot)
- `Target` (TargetInfo)
- `Result` (ResultInfo)
- `TraceID` (string)
- `RequestID` (string)
- `Detail` (typed: one of AuthnDetail, AuthzDetail, TupleMutationDetail, ResourceMutationDetail)

#### Scenario: Construct an authz decision event

- **WHEN** an AuditEvent is constructed with EventType=AuthzDecision, actor info, target "project:proj-123", and AuthzDetail with decision=allowed
- **THEN** all fields SHALL be populated and `Detail` SHALL be of type `*AuthzDetail`

#### Scenario: Construct a resource mutation event

- **WHEN** an AuditEvent is constructed with EventType=ResourceMutation for a "project" create operation
- **THEN** `Detail` SHALL be of type `*ResourceMutationDetail` containing the mutation type and affected resource

### Requirement: Emitter interface abstracts event sending

`pkg/audit` SHALL define an `Emitter` interface:
```
Emit(ctx context.Context, event *AuditEvent) error
Close() error
```

#### Scenario: BrokerEmitter sends to Kafka topic

- **WHEN** `Emit` is called on a `BrokerEmitter` with a valid AuditEvent
- **THEN** the event SHALL be proto-marshaled and published to the configured audit topic via `pkg/broker`

#### Scenario: LogEmitter writes to logger

- **WHEN** `Emit` is called on a `LogEmitter`
- **THEN** the event SHALL be serialized and written to the configured logger at Info level

#### Scenario: NoopEmitter discards silently

- **WHEN** `Emit` is called on a `NoopEmitter`
- **THEN** the call SHALL return nil without any side effect

### Requirement: Recorder aggregates and sends audit events

`pkg/audit` SHALL provide a `Recorder` that:
- Accepts an `Emitter` at construction
- Provides typed builder methods: `RecordAuthzDecision(...)`, `RecordTupleChange(...)`, `RecordResourceMutation(...)`, `RecordAuthnResult(...)`
- Populates common fields (EventID, OccurredAt, Service, TraceID, RequestID) automatically from context
- Calls `Emitter.Emit` with the fully constructed event

#### Scenario: Recorder auto-populates trace context

- **WHEN** `RecordAuthzDecision` is called with a context containing trace_id="abc-123" and request_id="req-456"
- **THEN** the emitted `AuditEvent` SHALL have TraceID="abc-123" and RequestID="req-456"

#### Scenario: Recorder auto-generates event ID

- **WHEN** any Record method is called
- **THEN** the emitted `AuditEvent.EventID` SHALL be a valid UUID

### Requirement: Audit middleware for Kratos

`pkg/audit` SHALL provide a Kratos middleware that can be composed into the server middleware chain. The middleware SHALL:
- Execute after the handler completes (post-handler position)
- Be configurable with audit rules per operation (similar to authz rules)
- In this skeleton phase, provide the middleware interface and option types without requiring codegen rules

#### Scenario: Middleware records event after handler success

- **WHEN** a request to operation "/order.v1.OrderService/CreateOrder" completes successfully
- **AND** an audit rule is configured for that operation
- **THEN** the middleware SHALL emit an AuditEvent with the operation, actor, and result=success

#### Scenario: Middleware records event after handler failure

- **WHEN** a request to an audited operation fails with an error
- **THEN** the middleware SHALL emit an AuditEvent with result=failure and the error information
