## ADDED Requirements

### Requirement: CreateTenant 必须提供一体化创建流程

系统必须提供 `CreateWithDefaults` 方法，一次调用完成：创建 Tenant → 创建默认 Organization → 创建默认 Project → 创建者成为 Tenant owner，并同步所有 FGA tuple。

#### Scenario: 一体化创建成功

- **WHEN** 调用 `TenantUsecase.CreateWithDefaults(ctx, &entity.Tenant{Slug: "acme", Name: "Acme Corp"}, creatorUserID)`
- **THEN** 创建 Tenant(slug=acme)、默认 Organization(tenant_id=新 tenant ID)、默认 Project(organization_id=新 org ID)，creator 成为 Tenant owner + Organization owner + Project admin（DB + FGA）

#### Scenario: Organization 创建失败回滚

- **WHEN** 调用 `CreateWithDefaults` 且默认 Organization 创建失败
- **THEN** 已创建的 Tenant 和 TenantMember 被回滚，已写入的 FGA tuple 被清理

#### Scenario: FGA 写入失败回滚

- **WHEN** 调用 `CreateWithDefaults` 且 Tenant owner FGA tuple 写入失败
- **THEN** DB 中的 TenantMember 被删除，Tenant 记录被删除

### Requirement: CreateTenant 必须仅限平台管理员调用

系统必须限制 CreateTenant API 仅允许拥有 `platform:default / can_manage_all_tenants` 权限的用户调用。

#### Scenario: 平台管理员创建 tenant

- **WHEN** 用户拥有 FGA tuple `user:{uid} → admin → platform:default`，调用 `POST /v1/tenants`
- **THEN** 请求被允许执行

#### Scenario: 普通用户被拒绝

- **WHEN** 用户不拥有 platform admin 权限，调用 `POST /v1/tenants`
- **THEN** 返回 403 权限不足
