## ADDED Requirements

### Requirement: Tenant 必须作为可 CRUD 的一等实体

系统必须提供 Tenant 实体的完整 CRUD 能力，包含 Ent schema、biz usecase、data repo、service 层。Tenant 必须包含 id、slug（unique）、name、domain（optional, unique）、kind（business/personal）、status（active/disabled）、created_at、updated_at、deleted_at 字段。

#### Scenario: 创建 business tenant

- **WHEN** 调用 `TenantUsecase.Create(ctx, &entity.Tenant{Slug: "acme", Name: "Acme Corp", Kind: "business"})`
- **THEN** 返回创建成功的 Tenant 实体，Kind 为 "business"，Status 为 "active"

#### Scenario: slug 唯一校验

- **WHEN** 已存在 slug="acme" 的 Tenant，再次调用 `Create` 传入 slug="acme"
- **THEN** 返回错误，提示 slug 已存在

#### Scenario: 通过 slug 查询

- **WHEN** 调用 `TenantRepo.GetBySlug(ctx, "acme")`
- **THEN** 返回 slug="acme" 的 Tenant 实体

#### Scenario: 软删除

- **WHEN** 调用 `TenantUsecase.Delete(ctx, tenantID)`
- **THEN** Tenant 的 deleted_at 被填充，不再出现在正常查询中

### Requirement: TenantMember 必须表达用户与租户的成员关系

系统必须提供 TenantMember 实体，包含 id、tenant_id、user_id、role（owner/admin/member）、status（active/invited）、joined_at、created_at、updated_at。tenant_id + user_id 必须唯一。

#### Scenario: 添加成员

- **WHEN** 调用 `TenantRepo.AddMember(ctx, &entity.TenantMember{TenantID: tid, UserID: uid, Role: "admin", Status: "active"})`
- **THEN** 返回创建成功的 TenantMember

#### Scenario: 重复添加

- **WHEN** tenant_id + user_id 组合已存在，再次调用 AddMember
- **THEN** 返回错误，提示成员已存在

#### Scenario: 更新角色

- **WHEN** 调用 `TenantRepo.UpdateMemberRole(ctx, tenantID, userID, "admin")`
- **THEN** 该成员的 role 更新为 "admin"

### Requirement: TenantUsecase 的 AddMember 和 RemoveMember 必须同步 FGA

系统必须在 TenantMember 的 DB 操作成功后同步写入或删除 FGA tuple。FGA 操作失败时必须回滚 DB 变更。

#### Scenario: AddMember FGA 同步

- **WHEN** 调用 `TenantUsecase.AddMember(ctx, tenantID, userID, "admin")`
- **THEN** DB 创建 TenantMember 记录，FGA 写入 tuple `user:{userID} → admin → tenant:{tenantID}`

#### Scenario: AddMember FGA 失败回滚

- **WHEN** 调用 `TenantUsecase.AddMember` 且 FGA WriteTuples 失败
- **THEN** DB 中的 TenantMember 记录被回滚删除

#### Scenario: RemoveMember FGA 同步

- **WHEN** 调用 `TenantUsecase.RemoveMember(ctx, tenantID, userID)`
- **THEN** DB 删除 TenantMember 记录，FGA 删除对应 tuple
