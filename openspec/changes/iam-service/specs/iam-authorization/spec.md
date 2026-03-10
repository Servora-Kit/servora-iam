# IAM Authorization Specification

## ADDED Requirements

### Requirement: The system MUST perform OpenFGA-based permission checking

系统 MUST 使用 OpenFGA 作为权限引擎，实现基于关系的访问控制（ReBAC）。

#### Scenario: Successful permission check

- **WHEN** 用户请求访问资源并提供 user_id、resource_type、resource_id 和 relation
- **THEN** 系统调用 OpenFGA Check API、返回是否有权限的布尔值

#### Scenario: Permission denied

- **WHEN** 用户没有访问资源的权限
- **THEN** 系统返回 `allowed: false`，HTTP 状态码 403

#### Scenario: Permission check with invalid resource

- **WHEN** 用户请求访问不存在的资源
- **THEN** 系统返回错误 "Resource not found"，HTTP 状态码 404

### Requirement: The system MUST support relation tuple management

系统 MUST 提供 API 用于管理 OpenFGA 关系元组（创建、删除）。

#### Scenario: Create relation tuple

- **WHEN** 管理员创建关系元组（如 "user:alice is member of workspace:ws1"）
- **THEN** 系统调用 OpenFGA Write API、创建元组、返回成功

#### Scenario: Delete relation tuple

- **WHEN** 管理员删除关系元组
- **THEN** 系统调用 OpenFGA Write API、删除元组、返回成功

#### Scenario: Create duplicate relation tuple

- **WHEN** 管理员尝试创建已存在的关系元组
- **THEN** 系统返回成功（幂等操作）

### Requirement: The system MUST enforce permission inheritance across tenant hierarchy

系统 MUST 支持三层权限继承（Platform → Tenant → Workspace），上层权限自动继承到下层。

#### Scenario: Tenant admin inherits workspace permissions

- **WHEN** 用户是 Tenant 的 admin
- **THEN** 用户自动拥有该 Tenant 下所有 Workspace 的 admin 权限

#### Scenario: Platform owner inherits all permissions

- **WHEN** 用户是 Platform 的 owner（V1 固定为 platform:root）
- **THEN** 用户自动拥有所有 Tenant 和 Workspace 的 owner 权限

### Requirement: The system MUST filter lists based on permissions

系统 MUST 支持基于权限的列表过滤，只返回用户有权限访问的资源。

#### Scenario: List workspaces with member permission

- **WHEN** 用户请求列出所有 Workspace
- **THEN** 系统调用 OpenFGA ListObjects、返回用户有 member 权限的 Workspace 列表

#### Scenario: Empty list when no permissions

- **WHEN** 用户没有任何 Workspace 的访问权限
- **THEN** 系统返回空列表

### Requirement: The system MUST use Redis-based pagination for list filtering

系统 MUST 使用 Redis 会话游标实现列表分页，第一次调用获取所有 ID 并存入 Redis，后续调用从 Redis 读取指定范围。

#### Scenario: First page request creates Redis session

- **WHEN** 用户首次请求列表（page=1, page_size=10）
- **THEN** 系统调用 OpenFGA StreamedListObjects 获取所有 ID、存入 Redis（key: `cursor:{sessionID}`, TTL 10 分钟）、返回前 10 个 ID 对应的资源

#### Scenario: Subsequent page request reads from Redis

- **WHEN** 用户请求第二页（page=2, page_size=10, cursor={sessionID}）
- **THEN** 系统从 Redis 读取 ID 列表、返回第 11-20 个 ID 对应的资源

#### Scenario: Cursor expires after TTL

- **WHEN** 用户在 10 分钟后使用过期的 cursor
- **THEN** 系统返回错误 "Cursor expired, please restart pagination"，HTTP 状态码 410

### Requirement: The system MUST cache permission checks with TTL

系统 MUST 使用 Redis 缓存权限检查结果，按接口敏感度区分 TTL。

#### Scenario: Cache hit for permission check

- **WHEN** 用户请求权限检查且 Redis 中存在缓存结果
- **THEN** 系统直接返回缓存结果，不调用 OpenFGA

#### Scenario: Cache miss triggers OpenFGA call

- **WHEN** 用户请求权限检查且 Redis 中无缓存
- **THEN** 系统调用 OpenFGA、将结果存入 Redis（敏感接口 TTL 1 分钟，普通接口 TTL 5 分钟）

#### Scenario: Cache invalidation on permission change

- **WHEN** 管理员修改用户权限（创建或删除关系元组）
- **THEN** 系统删除相关的 Redis 缓存

### Requirement: The system MUST provide authorization middleware for HTTP and gRPC

系统 MUST 提供权限中间件，自动检查用户是否有权限访问资源。

#### Scenario: Middleware allows authorized request

- **WHEN** 用户请求受保护的 API 且有权限
- **THEN** 中间件验证通过、请求继续处理

#### Scenario: Middleware blocks unauthorized request

- **WHEN** 用户请求受保护的 API 但无权限
- **THEN** 中间件返回错误 "Permission denied"，HTTP 状态码 403

#### Scenario: Middleware extracts resource ID from request

- **WHEN** API 路径包含资源 ID（如 `/v1/workspaces/{workspace_id}`）
- **THEN** 中间件自动提取 workspace_id 并进行权限检查

### Requirement: The system MUST fail closed when OpenFGA is unavailable

系统 MUST 在 OpenFGA 不可用时默认拒绝受保护请求，仅允许显式白名单端点（如健康检查）绕过鉴权。

#### Scenario: OpenFGA unavailable blocks protected requests

- **WHEN** OpenFGA 服务不可用且请求目标为受保护业务接口
- **THEN** 系统返回错误 "Authorization service unavailable"，HTTP 状态码 503

#### Scenario: Explicit allowlist endpoint bypass

- **WHEN** OpenFGA 服务不可用且请求目标为显式白名单端点
- **THEN** 系统允许请求通过并记录告警日志
