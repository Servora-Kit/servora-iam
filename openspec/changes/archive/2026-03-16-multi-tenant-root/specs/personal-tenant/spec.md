## ADDED Requirements

### Requirement: 系统必须在用户注册时自动创建 personal tenant

系统必须提供 `EnsurePersonalTenant(ctx, userID)` 方法。当用户首次注册或登录时，如果不存在 personal tenant，必须自动创建一个 kind=personal 的 Tenant 及其默认 Organization 和 Project。

#### Scenario: 首次注册创建 personal tenant

- **WHEN** 用户注册成功，调用 `EnsurePersonalTenant(ctx, userID)`，该用户不存在 kind=personal 的 Tenant
- **THEN** 创建 Tenant(kind=personal, slug=自动生成)、默认 Organization 和默认 Project，用户成为 personal tenant 的 owner

#### Scenario: 已有 personal tenant 直接返回

- **WHEN** 调用 `EnsurePersonalTenant(ctx, userID)`，该用户已存在 kind=personal 的 Tenant
- **THEN** 直接返回已有的 personal tenant，不创建新的

#### Scenario: 幂等性

- **WHEN** 并发调用两次 `EnsurePersonalTenant(ctx, userID)`
- **THEN** 只创建一个 personal tenant（unique constraint 或查询保证）

### Requirement: Personal tenant 禁止邀请其他成员

系统必须禁止向 kind=personal 的 Tenant 邀请其他成员。只有 owner（创建者）一人。

#### Scenario: 邀请被拒绝

- **WHEN** 调用 `TenantUsecase.InviteMember` 且目标 Tenant 的 kind=personal
- **THEN** 返回错误，提示 personal tenant 不允许邀请成员

### Requirement: Personal tenant 禁止删除

系统禁止删除 kind=personal 的 Tenant。只要用户存在，其 personal tenant 必须保留。

#### Scenario: 删除被拒绝

- **WHEN** 调用 `TenantUsecase.Delete(ctx, personalTenantID)`
- **THEN** 返回错误，提示 personal tenant 不允许删除
