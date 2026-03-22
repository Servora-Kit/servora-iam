## ADDED Requirements

### Requirement: Check and ListObjects accept full principal
`pkg/openfga` `Check` and `ListObjects` SHALL accept a `user string` parameter representing the full OpenFGA principal (e.g. `"user:uuid"`, `"service:id"`). The methods SHALL NOT prepend any type prefix internally.

#### Scenario: Check with user principal
- **WHEN** `Check(ctx, "user:abc-123", "viewer", "project", "proj-1")` is called
- **THEN** the OpenFGA SDK request SHALL contain `User: "user:abc-123"` without modification

#### Scenario: Check with service principal
- **WHEN** `Check(ctx, "service:api-gateway", "caller", "endpoint", "ep-1")` is called
- **THEN** the OpenFGA SDK request SHALL contain `User: "service:api-gateway"`

#### Scenario: ListObjects with full principal
- **WHEN** `ListObjects(ctx, "user:abc-123", "viewer", "project")` is called
- **THEN** the OpenFGA SDK request SHALL contain `User: "user:abc-123"` without modification

### Requirement: CachedCheck returns cache hit information
`CachedCheck` SHALL return `(allowed bool, cacheHit bool, err error)`. When the result is served from Redis cache, `cacheHit` SHALL be `true`. When the result is fetched from OpenFGA and stored in cache, `cacheHit` SHALL be `false`. When Redis client is nil (degraded mode), `cacheHit` SHALL be `false`.

#### Scenario: Cache hit
- **WHEN** `CachedCheck` is called and the result exists in Redis
- **THEN** the method SHALL return the cached result with `cacheHit = true`

#### Scenario: Cache miss
- **WHEN** `CachedCheck` is called and the result does not exist in Redis
- **THEN** the method SHALL call OpenFGA Check, store the result in Redis, and return with `cacheHit = false`

#### Scenario: No Redis client
- **WHEN** `CachedCheck` is called with `rdb = nil`
- **THEN** the method SHALL delegate to plain `Check` and return with `cacheHit = false`

### Requirement: NewClient accepts functional options
`NewClient` SHALL accept `(cfg *conf.App_OpenFGA, opts ...ClientOption)` and return `(*Client, error)`. `ClientOption` SHALL be a functional option type.

#### Scenario: NewClient with no options
- **WHEN** `NewClient(cfg)` is called without options
- **THEN** a Client SHALL be created with default settings (no recorder, no computed relations)

#### Scenario: NewClient with options
- **WHEN** `NewClient(cfg, WithAuditRecorder(r), WithComputedRelations(m))` is called
- **THEN** the Client SHALL hold the recorder and computed relation map for use in subsequent operations

### Requirement: Cache invalidation uses generic principal parsing
`parseTupleComponents` SHALL parse the User field of a Tuple as a generic `type:id` format, extracting the bare ID from any type prefix (not limited to `"user:"`).

#### Scenario: Parse user principal
- **WHEN** a Tuple with `User: "user:abc-123"` is processed for cache invalidation
- **THEN** the extracted ID SHALL be `"abc-123"`

#### Scenario: Parse service principal
- **WHEN** a Tuple with `User: "service:gateway"` is processed for cache invalidation
- **THEN** the extracted ID SHALL be `"gateway"`

#### Scenario: Parse organization member principal
- **WHEN** a Tuple with `User: "organization:org-1#member"` is processed for cache invalidation
- **THEN** the extracted ID SHALL be `"org-1#member"`

### Requirement: Computed relations are configurable via ClientOption
`affectedRelations` SHALL use a `ComputedRelationMap` held by the Client, injected via `WithComputedRelations(map[string][]string)`. When the map is nil or empty, cache invalidation SHALL only invalidate the tuple's own relation. The map SHALL NOT contain any hardcoded business-specific entries.

#### Scenario: Default computed relations (empty map)
- **WHEN** a Client is created without `WithComputedRelations`
- **AND** `InvalidateForTuples` is called for a tuple with relation `"admin"` on object type `"project"`
- **THEN** only cache entries for relation `"admin"` on `"project"` SHALL be invalidated

#### Scenario: Custom computed relations
- **WHEN** a Client is created with `WithComputedRelations(map[string][]string{"project": {"can_view", "can_edit"}})`
- **AND** `InvalidateForTuples` is called for a tuple with relation `"admin"` on object type `"project"`
- **THEN** cache entries for relations `"admin"`, `"can_view"`, and `"can_edit"` on `"project"` SHALL be invalidated
