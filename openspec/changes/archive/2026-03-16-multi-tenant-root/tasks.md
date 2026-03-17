# 任务：多租户根

## 第一阶段：FGA 模型扩展

- [x] **T1: FGA 模型扩展**
  - 文件：`manifests/openfga/model/servora.fga`
  - 新增 `platform` type（admin、can_view_all_tenants、can_manage_all_tenants）
  - tenant 增加 `owner` relation，层级继承 owner → admin → member
  - tenant 增加 `can_manage_members: owner or admin`
  - organization 的 `admin` 和 `member` 增加 `from tenant` 继承
  - 执行 `make openfga.model.validate` 验证
  - 更新 `manifests/openfga/model/servora.fga.tests`（如有）并执行 `make openfga.model.test`

## 第二阶段：Ent Schema 与生成

- [x] **T2: 扩展 Tenant Schema**
  - 文件：`internal/data/schema/tenant.go`
  - 删除 `type` 字段，新增 `kind`（enum: business/personal）、`domain`（optional, unique）、`status`（enum: active/disabled）、`updated_at`
  - 加入 `SoftDeleteMixin`
  - 新增 Edge：members → TenantMember
  - 执行 `make gen.ent` 验证

- [x] **T3: 新建 TenantMember Schema**
  - 文件：`internal/data/schema/tenant_member.go`
  - 字段：id、tenant_id、user_id、role（enum: owner/admin/member）、status（enum: active/invited）、joined_at、created_at、updated_at
  - Edge：tenant ← Tenant、user ← User（cascade delete）
  - 索引：unique(tenant_id, user_id)
  - User schema 增加 `tenant_members` Edge
  - 执行 `make gen.ent` 验证

- [x] **T4: OrganizationMember/ProjectMember 增加 status 字段**
  - 文件：`internal/data/schema/organization_member.go`、`internal/data/schema/project_member.go`
  - 新增 `status` 字段：enum("active", "invited")，default "active"
  - 执行 `make gen.ent` 验证

## 第三阶段：Entity + Biz + Data

- [x] **T5: 新建 Tenant Entity**
  - 文件：`internal/biz/entity/tenant.go`
  - Tenant 和 TenantMember 结构体

- [x] **T6: OrganizationMember/ProjectMember Entity 增加 Status 字段**
  - 文件：`internal/biz/entity/organization.go`、`internal/biz/entity/project.go`
  - OrganizationMember 和 ProjectMember 结构体增加 `Status string`

- [x] **T7: 新建 TenantRepo 接口与 TenantUsecase**
  - 文件：`internal/biz/tenant.go`
  - TenantRepo 接口（CRUD + Member 管理 + GetPersonalTenantByUserID）
  - TenantUsecase：Create（含 FGA 回滚）、CreateWithDefaults（一体化）、EnsurePersonalTenant、AddMember/RemoveMember/UpdateMemberRole
  - InviteMember/AcceptInvitation/RejectInvitation
  - Personal tenant 约束（禁止邀请、禁止删除）

- [x] **T8: 新建 TenantRepo Data 实现**
  - 文件：`internal/data/tenant.go`
  - 实现 T7 定义的接口，使用 Ent Client

## 第四阶段：去除 TenantRootID 硬编码 + Tenant Scope

- [x] **T9: OrganizationUsecase 去除固定 tenantID**
  - 文件：`internal/biz/organization.go`
  - 去除 `tenantID string` 字段和 `TenantRootID` 构造参数
  - Create/CreateDefault 的 tenantID 改为从调用参数传入
  - InviteMember/AcceptInvitation/RejectInvitation 方法

- [x] **T10: UserUsecase 适配多 tenant**
  - 文件：`internal/biz/user.go`
  - 去除 `tenantID string` 字段，改为依赖 TenantRepo
  - collectUserFGATuples 和 CompensateUserPurge 中 tenant tuple 改为遍历 ListMembershipsByUserID

- [x] **T11: ProjectUsecase 邀请方法**
  - 文件：`internal/biz/project.go`
  - 新增 InviteMember/AcceptInvitation/RejectInvitation 方法

- [x] **T12: Organization 查询按 tenant_id 过滤**
  - 文件：`internal/data/organization.go`
  - ListByUserID 增加 tenantID 参数
  - data 层加 `Where(organization.TenantIDEQ(tid))`

- [x] **T13: Actor scope 扩展 TenantID**
  - 文件：`pkg/actor/actor.go`
  - Actor 结构体增加 `TenantID` 字段
  - 从 X-Tenant-ID header 解析并注入

- [x] **T14: Authz 中间件去除 WithTenantRootID**
  - 文件：`internal/server/middleware/authz.go`
  - `AUTHZ_MODE_OBJECT` + `tenant` 类型改为从请求提取 tenant_id
  - 去除 `authzConfig.tenantRootID` 和 `WithTenantRootID` 选项
  - 更新 `internal/server/grpc.go`、`http.go` 的中间件注册

## 第五阶段：Proto + Service + Wire

- [x] **T15: 新建 Tenant Proto**
  - 文件：`api/protos/tenant/service/v1/tenant.proto`（共享消息）
  - 文件：`app/iam/service/api/protos/iam/service/v1/i_tenant.proto`（IAM 路由）
  - 消息：CreateTenantRequest/Response、GetTenantRequest、ListTenantsRequest、InviteTenantMemberRequest、AcceptInvitationRequest 等
  - 包含 authz 注解
  - 执行 `make api` 生成

- [x] **T16: 新建 TenantService**
  - 文件：`internal/service/tenant.go`
  - 实现 T15 定义的 gRPC service 接口
  - 注册到 service.ProviderSet

- [x] **T17: Wire 与 Server 注册**
  - 更新 `internal/data/data.go` ProviderSet：NewTenantRepo
  - 更新 `internal/server/grpc.go`、`http.go`：注册 TenantService
  - 更新 `cmd/server/wire.go`：新增 Tenant 相关 Provider
  - 去除 NewTenantRootID
  - 执行 `make wire`

## 第六阶段：Seed 调整与验证

- [x] **T18: Seed 调整**
  - 文件：`internal/data/seed.go`
  - Seed `platform:default` admin tuple（替代原 tenant:root admin）
  - 保留初始 tenant 创建（slug=default），但不再注入为全局常量
  - seedTenantAdminFGA 改为 owner tuple

- [x] **T19: 构建与测试**
  - `go build ./app/iam/service/...`
  - 运行现有单测，确保无回归
  - 新增 TenantUsecase 单测（Create/CreateWithDefaults/EnsurePersonalTenant/AddMember/FGA 回滚/InviteMember/AcceptInvitation）
  - 新增 FGA model test case
