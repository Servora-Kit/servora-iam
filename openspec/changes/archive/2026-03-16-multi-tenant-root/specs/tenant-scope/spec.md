## ADDED Requirements

### Requirement: 系统必须支持请求级 Tenant 上下文注入

系统必须从请求的 `X-Tenant-ID` header 解析 tenant ID 并注入 actor scope，与现有 X-Organization-ID 模式一致。

#### Scenario: X-Tenant-ID header 存在

- **WHEN** 请求携带 `X-Tenant-ID: {tenant-uuid}` header
- **THEN** `actor.FromContext(ctx).TenantID` 返回该 UUID

#### Scenario: X-Tenant-ID header 缺失

- **WHEN** 请求未携带 `X-Tenant-ID` header
- **THEN** `actor.FromContext(ctx).TenantID` 返回空字符串

### Requirement: Organization 查询必须按 tenant_id 过滤

系统在 tenant 上下文存在时，必须对 Organization 查询附加 `tenant_id` 过滤条件（defense-in-depth），防止跨租户数据泄露。

#### Scenario: ListByUserID 带 tenantID 过滤

- **WHEN** 调用 `OrganizationRepo.ListByUserID(ctx, userID, tenantID, page, pageSize)` 且 tenantID 非空
- **THEN** 返回的 Organization 列表中所有记录的 tenant_id 均等于传入的 tenantID

#### Scenario: ListByUserID 不带 tenantID

- **WHEN** 调用 `OrganizationRepo.ListByUserID(ctx, userID, "", page, pageSize)` 且 tenantID 为空
- **THEN** 返回用户有权访问的所有 Organization（不按 tenant 过滤）

### Requirement: 系统必须去除 TenantRootID 硬编码依赖

系统禁止通过 Wire 注入固定的 TenantRootID。OrganizationUsecase 和 UserUsecase 必须从参数或 TenantRepo 动态获取 tenant 信息。

#### Scenario: OrganizationUsecase 不再持有固定 tenantID

- **WHEN** 构造 `NewOrganizationUsecase`
- **THEN** 构造函数参数中不包含 `TenantRootID`，`Create` 方法的 tenantID 从调用参数传入

#### Scenario: UserUsecase 适配多 tenant

- **WHEN** 调用 `collectUserFGATuples(ctx, userID)`
- **THEN** tenant 相关 tuple 通过 `TenantRepo.ListMembershipsByUserID` 动态获取，而非使用固定 tenantID

#### Scenario: Authz 中间件不使用 WithTenantRootID

- **WHEN** 配置 Authz 中间件
- **THEN** 不存在 `WithTenantRootID` 选项，`AUTHZ_MODE_OBJECT` + `tenant` 类型从请求字段提取 tenant_id
