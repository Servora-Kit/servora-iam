## Purpose

Define how `pkg/openfga` tuple mutation operations (`WriteTuples`, `DeleteTuples`) automatically emit audit events. The design uses core/public method layering to separate pure SDK operations from cross-cutting concerns, enabling clean extensibility for future additions (metrics, tracing).

## Requirements

### Requirement: WriteTuples emits tuple.changed audit event on success
`WriteTuples` SHALL call `recorder.RecordTupleChange` after a successful write operation. The event SHALL include the operation name (`"openfga.WriteTuples"`), the actor from context, and a `TupleMutationDetail` with mutation type `"write"` and the list of written tuples. When the recorder is nil, no event SHALL be emitted.

#### Scenario: Successful write emits event
- **WHEN** `WriteTuples(ctx, tuple1, tuple2)` succeeds
- **AND** the Client has an audit recorder configured
- **THEN** the recorder SHALL receive a `RecordTupleChange` call with mutation type `"write"` and both tuples

#### Scenario: Failed write does not emit
- **WHEN** `WriteTuples(ctx, tuple1)` returns an error from the OpenFGA SDK
- **THEN** no audit event SHALL be emitted

#### Scenario: No recorder skips emission
- **WHEN** `WriteTuples(ctx, tuple1)` succeeds
- **AND** the Client has no audit recorder (nil)
- **THEN** the write SHALL succeed without any audit emission

### Requirement: DeleteTuples emits tuple.changed audit event on success
`DeleteTuples` SHALL call `recorder.RecordTupleChange` after a successful delete operation. The event SHALL include the operation name (`"openfga.DeleteTuples"`), the actor from context, and a `TupleMutationDetail` with mutation type `"delete"` and the list of deleted tuples. When the recorder is nil, no event SHALL be emitted.

#### Scenario: Successful delete emits event
- **WHEN** `DeleteTuples(ctx, tuple1)` succeeds
- **AND** the Client has an audit recorder configured
- **THEN** the recorder SHALL receive a `RecordTupleChange` call with mutation type `"delete"` and the tuple

#### Scenario: Failed delete does not emit
- **WHEN** `DeleteTuples(ctx, tuple1)` returns an error
- **THEN** no audit event SHALL be emitted

### Requirement: Tuple operations use core/public layering
`WriteTuples` and `DeleteTuples` SHALL be structured as public wrapper methods that compose an unexported core method (pure SDK operation) with cross-cutting concerns (audit emit). The core methods (`writeTuplesCore`, `deleteTuplesCore`) SHALL contain only the OpenFGA SDK call logic.

#### Scenario: Core method isolation
- **WHEN** the internal `writeTuplesCore` method is called
- **THEN** it SHALL only perform the OpenFGA SDK write operation without any side effects

#### Scenario: Public method composition
- **WHEN** the public `WriteTuples` method is called
- **THEN** it SHALL call `writeTuplesCore` first, then conditionally emit an audit event on success

### Requirement: Audit event includes actor from context
Tuple audit events SHALL extract the actor from the request context using `actor.FromContext(ctx)`. If no actor is present in the context, the event SHALL still be emitted with a zero-value actor info.

#### Scenario: Actor present in context
- **WHEN** `WriteTuples` is called with a context containing a user actor
- **THEN** the audit event SHALL include the actor's ID, type, and display name

#### Scenario: No actor in context
- **WHEN** `WriteTuples` is called with a context that has no actor
- **THEN** the audit event SHALL be emitted with empty actor info
