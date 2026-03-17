## MODIFIED Requirements

### Requirement: 所有 Member 实体必须统一包含 status 字段

系统必须在 TenantMember、OrganizationMember、ProjectMember 三个实体上统一增加 `status` 字段，值为 `active` 或 `invited`，默认 `active`。

#### Scenario: 现有 AddMember 默认 active

- **WHEN** 调用 `OrganizationUsecase.AddMember(ctx, orgID, userID, "member")` 直接添加成员
- **THEN** 创建的 OrganizationMember 的 status 为 "active"

#### Scenario: 通过邀请添加成员

- **WHEN** 调用 `OrganizationUsecase.InviteMember(ctx, orgID, userID, "member")`
- **THEN** 创建的 OrganizationMember 的 status 为 "invited"

## ADDED Requirements

### Requirement: 系统必须提供邀请到接受流程

系统必须提供 InviteMember 和 AcceptInvitation 方法。InviteMember 创建 status=invited 的成员记录并预写 FGA tuple。AcceptInvitation 将 status 更新为 active 并填充 joined_at。

#### Scenario: 邀请成员

- **WHEN** 调用 `TenantUsecase.InviteMember(ctx, tenantID, userID, "member")`
- **THEN** 创建 TenantMember(status=invited, role=member)，FGA 写入 `user:{userID} → member → tenant:{tenantID}` tuple

#### Scenario: 接受邀请

- **WHEN** 被邀请用户调用 `TenantUsecase.AcceptInvitation(ctx, tenantID, userID)`
- **THEN** TenantMember 的 status 更新为 "active"，joined_at 被填充为当前时间

#### Scenario: 拒绝邀请

- **WHEN** 被邀请用户调用 `TenantUsecase.RejectInvitation(ctx, tenantID, userID)`
- **THEN** TenantMember 记录被删除，FGA 中对应 tuple 被删除

#### Scenario: 重复接受

- **WHEN** TenantMember 的 status 已经是 "active"，再次调用 AcceptInvitation
- **THEN** 操作幂等，不报错

### Requirement: OrganizationUsecase 和 ProjectUsecase 必须提供同等的邀请流程

系统必须在 OrganizationUsecase 和 ProjectUsecase 中提供与 TenantUsecase 一致的 InviteMember/AcceptInvitation/RejectInvitation 方法。

#### Scenario: Organization 邀请成员

- **WHEN** 调用 `OrganizationUsecase.InviteMember(ctx, orgID, userID, "member")`
- **THEN** 创建 OrganizationMember(status=invited)，FGA 写入对应 tuple

#### Scenario: Project 邀请成员

- **WHEN** 调用 `ProjectUsecase.InviteMember(ctx, orgID, projID, userID, "member")`
- **THEN** 创建 ProjectMember(status=invited)，FGA 写入对应 tuple
