# Spec: infra-kafka-clickhouse

## Purpose

Defines requirements for the `infra-kafka-clickhouse` capability.

## Requirements

### Requirement: Kafka service in docker-compose

`docker-compose.yaml` SHALL include a Kafka service using `apache/kafka:latest` in KRaft mode (no ZooKeeper), with:
- Container name: `servora_kafka`
- Internal listener on port 9092
- Exposed port mapping: `9092:9092`
- Health check using kafka-topics.sh
- Named volume for data persistence
- Connected to `servora-network`

Configuration SHALL reference the Kemate Kafka compose pattern for KRaft environment variables.

#### Scenario: Kafka starts and becomes healthy

- **WHEN** `docker compose up kafka` is run
- **THEN** the Kafka container SHALL start in KRaft mode, pass health check, and accept connections on port 9092

### Requirement: ClickHouse service in docker-compose

`docker-compose.yaml` SHALL include a ClickHouse service using `clickhouse/clickhouse-server:latest`, with:
- Container name: `servora_clickhouse`
- HTTP interface on port 8123 (mapped to host 18123)
- Native interface on port 9000 (mapped to host 19000)
- Health check using clickhouse-client
- Named volume for data persistence
- Connected to `servora-network`

No table schemas SHALL be created in this phase.

#### Scenario: ClickHouse starts and becomes healthy

- **WHEN** `docker compose up clickhouse` is run
- **THEN** the ClickHouse container SHALL start, pass health check, and accept queries

### Requirement: IAM and sayhello removed from dev toolchain

The following SHALL be removed or disabled:
- `docker-compose.dev.yaml`: remove IAM and sayhello service entries (if present)
- `Makefile`: remove or comment out targets that start/build IAM and sayhello for development
- `app.mk`: ensure no hardcoded references to iam/sayhello remain in shared make targets

Service source code in `app/iam/service/` and `app/sayhello/service/` SHALL remain untouched.

#### Scenario: make compose.dev does not start IAM or sayhello

- **WHEN** `make compose.dev` is run
- **THEN** neither IAM nor sayhello containers SHALL be started

#### Scenario: Service code remains compilable

- **WHEN** `go build ./app/iam/service/...` is run
- **THEN** compilation SHALL succeed (service code is preserved, only toolchain references removed)
