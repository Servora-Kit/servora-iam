# Spec: broker-abstraction

## Purpose

Defines requirements for the `broker-abstraction` capability.

## Requirements

### Requirement: Broker interface defines publish-subscribe lifecycle

`pkg/broker` SHALL define a `Broker` interface with the following methods:
- `Connect(ctx context.Context) error`
- `Disconnect(ctx context.Context) error`
- `Publish(ctx context.Context, topic string, msg *Message, opts ...PublishOption) error`
- `Subscribe(ctx context.Context, topic string, handler Handler, opts ...SubscribeOption) (Subscriber, error)`

#### Scenario: Broker lifecycle — connect and disconnect

- **WHEN** `Connect` is called on a properly configured broker
- **THEN** the broker SHALL establish connection to the underlying messaging system and return nil error

#### Scenario: Broker publish sends message

- **WHEN** `Publish` is called with a valid topic and message
- **THEN** the message SHALL be delivered to the underlying messaging system

### Requirement: Message structure carries key, headers, and body

`Message` SHALL be a struct with fields:
- `Key string` — partition key / routing key
- `Headers Headers` — metadata key-value pairs (type alias for `map[string]string`)
- `Body []byte` — serialized payload

#### Scenario: Message with all fields

- **WHEN** a `Message` is constructed with Key="user-123", Headers={"trace_id": "abc"}, Body=[]byte{...}
- **THEN** all fields SHALL be accessible via struct field access

### Requirement: Event interface for consumed messages

`Event` interface SHALL provide:
- `Topic() string`
- `Message() *Message`
- `Ack() error`
- `Nack() error`

#### Scenario: Consumer acknowledges event

- **WHEN** a handler receives an Event and calls `Ack()`
- **THEN** the underlying system SHALL mark the message as consumed

### Requirement: Subscriber interface for unsubscription

`Subscriber` interface SHALL provide:
- `Topic() string`
- `Options() SubscribeOptions`
- `Unsubscribe(removeFromManager bool) error`

#### Scenario: Unsubscribe stops consumption

- **WHEN** `Unsubscribe()` is called on an active subscriber
- **THEN** the handler SHALL stop receiving new events from that topic

### Requirement: Handler type definition

`Handler` SHALL be defined as `func(ctx context.Context, event Event) error`.

#### Scenario: Handler receives context with trace

- **WHEN** a message with trace headers is consumed
- **THEN** the handler's context SHALL contain the propagated trace information

### Requirement: Kafka implementation of Broker

`pkg/broker/kafka` SHALL implement the `Broker` interface using franz-go.

The Kafka broker SHALL support:
- KRaft mode (no ZooKeeper dependency)
- Sync producer with configurable acks
- Consumer group with configurable group ID
- OTel trace propagation via message headers

#### Scenario: Kafka publish and consume round-trip

- **WHEN** a message is published to topic "test.events" via Kafka broker
- **AND** a subscriber is listening on the same topic
- **THEN** the subscriber's handler SHALL receive the message with matching key, headers, and body

#### Scenario: Kafka broker configuration

- **WHEN** a Kafka broker is created with brokers=["kafka:9092"], groupID="audit-consumer"
- **THEN** the broker SHALL connect to the specified Kafka cluster and use the given consumer group
