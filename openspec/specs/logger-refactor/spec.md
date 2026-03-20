# Spec: logger-refactor

> 对应 design.md D0

## Purpose

Defines requirements for the refactored `pkg/logger` API. The goal is to simplify
logger construction and usage so callers are as ergonomic as possible.

## Requirements

### Requirement: New constructor accepts proto config directly

`logger.New(app *conf.App) *ZapLogger` SHALL accept the proto `App` config directly,
reading `app.Env` and `app.GetLog()` without requiring a separate `Config` struct.

- `New(nil)` SHALL NOT panic and SHALL return a dev-mode console logger with sensible defaults.
- The return type SHALL be `*ZapLogger` (concrete type), enabling callers to access `Zap()` and `Sync()`.
- `logger.Config` struct and `NewLogger` function SHALL be removed.

#### Scenario: New(nil) is safe

- **WHEN** `logger.New(nil)` is called
- **THEN** it SHALL return a non-nil `*ZapLogger` using dev-mode defaults without panicking

#### Scenario: New(app) reads config fields

- **WHEN** `logger.New(app)` is called with a valid `App` proto containing `Env` and log level
- **THEN** the returned logger SHALL respect those settings

### Requirement: For shorthand creates module helpers in one line

`logger.For(l log.Logger, module string) *log.Helper` SHALL create a module-tagged
helper in a single call, equivalent to `logger.NewHelper(l, logger.WithModule(module))`.

All existing `NewHelper(l, WithModule("..."))` call sites SHALL be migrated to `For`.

#### Scenario: For returns a Helper

- **WHEN** `logger.For(l, "user/biz")` is called
- **THEN** it SHALL return a `*log.Helper` whose log entries carry `module=user/biz`

### Requirement: With accepts module string shorthand

`logger.With(l log.Logger, args ...any)` SHALL accept a bare `string` as the first
extra argument and treat it as the module name — equivalent to passing `WithModule(s)`.

The Option-style variant (`With(l, WithModule("..."))`) SHALL remain supported for
backward compatibility and multi-option calls.

#### Scenario: With string shorthand

- **WHEN** `logger.With(l, "http/server")` is called
- **THEN** the resulting logger SHALL tag entries with `module=http/server`

### Requirement: Zap() getter exposes underlying zap.Logger

`*ZapLogger` SHALL expose a `Zap() *zap.Logger` public method.

Adapter functions (`GetGormLogger`, `EntLogFuncFrom`) SHALL use `Zap()` instead of
accessing the unexported `log` field directly. Downstream packages (e.g. `pkg/broker`
kafka kzap plugin) SHALL obtain `*zap.Logger` via this getter.

#### Scenario: Zap getter is accessible

- **WHEN** `zapLogger.Zap()` is called on a `*ZapLogger`
- **THEN** it SHALL return the underlying non-nil `*zap.Logger`

### Requirement: Sync is a method, not a field

The exported `Sync func() error` field SHALL be removed and replaced by a `Sync() error`
method that delegates to `l.log.Sync()`. No external callers are impacted (field was never
used externally).

#### Scenario: Sync method succeeds

- **WHEN** `zapLogger.Sync()` is called
- **THEN** it SHALL return nil or a non-fatal error without panicking

### Requirement: Internal buildCore eliminates duplication

The package SHALL provide an internal `buildCore(env, level string, writers ...zapcore.WriteSyncer) zapcore.Core`
helper that eliminates the ~20-line prod/default duplication in `New`. The function SHALL
NOT be exported; it is an implementation detail of the `New` constructor.

#### Scenario: buildCore is not exported

- **WHEN** `go build ./pkg/logger/...` succeeds
- **THEN** no exported symbol named `buildCore` SHALL appear in the package API

### Requirement: All call sites SHALL be migrated to new API

The following SHALL be migrated:
- `pkg/bootstrap/bootstrap.go`: `NewLogger(&Config{...})` → `New(bc.App)`
- `app/iam/service/**/*.go`: `NewHelper(l, WithModule("x"))` → `For(l, "x")`
- `app/sayhello/service/**/*.go`: same migration
- `pkg/redis`, `pkg/openfga`, `pkg/transport/**`, `pkg/jwks`, `pkg/governance`: same migration
- Module names SHALL drop the `-service` suffix (e.g. `"user/data/iam-service"` → `"user/data/iam"`)

#### Scenario: Full project compiles after migration

- **WHEN** `go build ./...` and `go test ./pkg/logger/...` are run
- **THEN** both SHALL succeed with zero errors

### Requirement: Tests SHALL cover new API surface

Test files SHALL include:
- `New(nil)` safety test
- `For` helper test
- `Zap()` getter test
- `Sync()` method test
- Existing `log_defaults_test.go` and `gorm_log_test.go` SHALL be updated to the new API

#### Scenario: Test suite passes after migration

- **WHEN** `go test ./pkg/logger/...` is run after the migration
- **THEN** all tests SHALL pass with zero failures
