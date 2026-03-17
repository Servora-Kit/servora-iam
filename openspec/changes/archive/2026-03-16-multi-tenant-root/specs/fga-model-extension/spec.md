## ADDED Requirements

### Requirement: FGA 模型必须引入独立的 platform type

系统必须在 FGA 模型中新增 `platform` type，包含 `admin` relation 和 `can_view_all_tenants`、`can_manage_all_tenants` 权限。平台管理员必须通过 `platform:default` 鉴权，而非复用 `tenant:root`。

#### Scenario: platform admin 权限检查

- **WHEN** 用户拥有 FGA tuple `user:{uid} → admin → platform:default`，执行 `Check(user:{uid}, can_manage_all_tenants, platform:default)`
- **THEN** 返回 allowed=true

#### Scenario: 普通 tenant admin 无 platform 权限

- **WHEN** 用户仅拥有 `user:{uid} → admin → tenant:{tid}`，执行 `Check(user:{uid}, can_manage_all_tenants, platform:default)`
- **THEN** 返回 allowed=false

### Requirement: tenant FGA type 必须使用层级角色模型

系统必须在 tenant FGA type 中定义层级角色：`owner ⊃ admin ⊃ member`。owner 自动继承 admin 权限，admin 自动继承 member 权限。

#### Scenario: owner 自动拥有 admin 权限

- **WHEN** 用户拥有 FGA tuple `user:{uid} → owner → tenant:{tid}`，执行 `Check(user:{uid}, admin, tenant:{tid})`
- **THEN** 返回 allowed=true

#### Scenario: owner 自动拥有 can_view 权限

- **WHEN** 用户拥有 FGA tuple `user:{uid} → owner → tenant:{tid}`，执行 `Check(user:{uid}, can_view, tenant:{tid})`
- **THEN** 返回 allowed=true

#### Scenario: member 无 can_manage 权限

- **WHEN** 用户仅拥有 `user:{uid} → member → tenant:{tid}`，执行 `Check(user:{uid}, can_manage, tenant:{tid})`
- **THEN** 返回 allowed=false

### Requirement: organization 权限必须从 tenant 继承

系统必须在 organization FGA type 的 `admin` 和 `member` relation 中增加 `from tenant` 继承。tenant admin 必须自动获得其下所有 organization 的 admin 权限。

#### Scenario: tenant admin 自动拥有 organization admin 权限

- **WHEN** 用户拥有 `user:{uid} → admin → tenant:{tid}`，organization 拥有 `tenant:{tid} → tenant → organization:{oid}`，执行 `Check(user:{uid}, admin, organization:{oid})`
- **THEN** 返回 allowed=true

#### Scenario: tenant member 自动拥有 organization member 权限

- **WHEN** 用户拥有 `user:{uid} → member → tenant:{tid}`，organization 拥有 `tenant:{tid} → tenant → organization:{oid}`，执行 `Check(user:{uid}, member, organization:{oid})`
- **THEN** 返回 allowed=true

### Requirement: Seed 必须创建 platform default admin tuple

系统在初始化时必须 seed `platform:default` 并为初始管理员写入 `admin` tuple，替代原有的 `tenant:root` admin seed。

#### Scenario: 初始化后 platform admin 可用

- **WHEN** 执行 seed 流程
- **THEN** FGA 中存在 `user:{seed-admin-id} → admin → platform:default` tuple
