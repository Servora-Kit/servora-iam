# 设计：多租户根

参照 Kemate（`app/kemate/service/internal/data/schema/tenant.go`、`biz/tenant.go`、`biz/workspace.go`、`openfga/tuples.go`、`manifests/openfga/model/kemate.fga`）、go-wind-admin、Casdoor 的多租户与 RBAC 设计。

---

## 设计决策

经对比 Kemate（B2B2C + FGA）、go-wind-admin（Casbin + 角色模板）、Casdoor（Casbin + 单 org 限制），确定以下架构决策：

| # | 决策点 | 选择 | 理由 |
|---|--------|------|------|
| 1 | Platform type | 引入独立 FGA `platform` type | B2B+B2C 需明确区分平台管理 vs 租户管理；任何 tenant 可删除 |
| 2 | FGA 角色模型 | 层级（owner ⊃ admin ⊃ member） | 与现有 org/project 一致；authz 中间件可直接 Check role 名 |
| 3 | Member status | 加 status（active/invited） | 支持邀请流程；三层 Member 保持一致 |
| 4 | Personal tenant | 注册时自动创建（EnsurePersonalTenant） | B2C 需要；personal tenant 只有 owner 自己 |
| 5 | Tenant scope | middleware（X-Tenant-ID） | 与现有 X-Organization-ID 模式一致 |
| 6 | Slug | 保留（Tenant + Organization 一致） | URL 友好路径 |
| 7 | 软删除 | deleted_at（SoftDeleteMixin） | 与现有 Organization/Project/User 实体一致 |

---

## 1. FGA 模型扩展

### 1.1 引入 platform type

新增独立的 `platform` FGA type，取代 `tenant:root` 充当平台管理员的做法：

```
type platform
  relations
    define admin: [user]
    define can_view_all_tenants: admin
    define can_manage_all_tenants: admin
```

Seed 时创建 `platform:default` 并写入管理员 tuple。CreateTenant 权限改为 `Check(user, can_manage_all_tenants, platform, default)`。

### 1.2 tenant 扩展

```
type tenant
  relations
    define owner: [user]
    define admin: [user] or owner
    define member: [user] or admin
    define can_view: member
    define can_manage: admin
    define can_manage_members: owner or admin
```

新增 `owner` relation，层级继承：owner → admin → member。

### 1.3 organization 权限继承

```
type organization
  relations
    define tenant: [tenant]
    define owner: [user]
    define admin: [user] or owner or admin from tenant
    define member: [user] or admin or member from tenant
    define viewer: [user] or member
    define can_view: member or viewer
    define can_manage: admin
    define can_manage_members: admin
```

organization 的 `admin` 和 `member` 增加 `from tenant` 继承——tenant admin 自动获得其下所有 organization 的对应权限。

### 1.4 project 保持不变

project 已通过 `from organization` 继承权限，不需要额外修改。tenant 权限通过 organization 传递。

---

## 2. Ent Schema

### 2.1 Tenant Schema 扩展

现有 `schema/tenant.go` 字段较少（id/slug/name/type/created_at），扩展为：

```
字段变更：
- type: 删除（与 kind 语义重叠）
- kind: enum("business", "personal"), default "business"（取代原 type 字段）
- domain: string, optional, max 128, unique（自定义域名/入口标识）
- status: enum("active", "disabled"), default "active"
- updated_at: time, 自动更新

Mixin：
- SoftDeleteMixin（加入 deleted_at 字段）

Edge 新增：
- members → TenantMember（1:N）
```

### 2.2 TenantMember Schema（新建）

```
字段：
- id: UUID v7
- tenant_id: UUID（FK → Tenant）
- user_id: UUID（FK → User）
- role: enum("owner", "admin", "member"), default "member"
- status: enum("active", "invited"), default "active"
- joined_at: time, optional（接受邀请时填充）
- created_at / updated_at

Edge：
- tenant ← Tenant（M:1, required, cascade delete）
- user ← User（M:1, required, cascade delete）

索引：
- unique(tenant_id, user_id)
```

### 2.3 OrganizationMember / ProjectMember 补充 status 字段

为保持三层 Member 一致性，已有的 OrganizationMember 和 ProjectMember schema 各增加：

```
- status: enum("active", "invited"), default "active"
```

业务含义：`invited` 表示已发出邀请但未接受；`active` 表示已接受或直接添加。

### 2.4 User Schema 补充 Edge

User schema 新增 `tenant_members` edge，反向关联 TenantMember：

```go
edge.To("tenant_members", TenantMember.Type)
```

---

## 3. Biz 层

### 3.1 Entity

```go
type Tenant struct {
    ID        string
    Slug      string
    Name      string
    Domain    string
    Kind      string  // "business" | "personal"
    Status    string  // "active" | "disabled"
    CreatedAt time.Time
    UpdatedAt time.Time
}

type TenantMember struct {
    ID        string
    TenantID  string
    UserID    string
    UserName  string
    UserEmail string
    Role      string  // "owner" | "admin" | "member"
    Status    string  // "active" | "invited"
    JoinedAt  *time.Time
    CreatedAt time.Time
}
```

### 3.2 TenantRepo 接口

```go
type TenantRepo interface {
    Create(ctx context.Context, t *entity.Tenant) (*entity.Tenant, error)
    GetByID(ctx context.Context, id string) (*entity.Tenant, error)
    GetBySlug(ctx context.Context, slug string) (*entity.Tenant, error)
    GetByDomain(ctx context.Context, domain string) (*entity.Tenant, error)
    List(ctx context.Context, userID string, page, pageSize int32) ([]*entity.Tenant, int64, error)
    Update(ctx context.Context, t *entity.Tenant) (*entity.Tenant, error)
    Delete(ctx context.Context, id string) error
    Purge(ctx context.Context, id string) error

    AddMember(ctx context.Context, m *entity.TenantMember) (*entity.TenantMember, error)
    RemoveMember(ctx context.Context, tenantID, userID string) error
    GetMember(ctx context.Context, tenantID, userID string) (*entity.TenantMember, error)
    ListMembers(ctx context.Context, tenantID string, page, pageSize int32) ([]*entity.TenantMember, int64, error)
    UpdateMemberRole(ctx context.Context, tenantID, userID, role string) (*entity.TenantMember, error)
    UpdateMemberStatus(ctx context.Context, tenantID, userID, status string) (*entity.TenantMember, error)
    ListMembershipsByUserID(ctx context.Context, userID string) ([]*entity.TenantMember, error)
    GetPersonalTenantByUserID(ctx context.Context, userID string) (*entity.Tenant, error)
}
```

### 3.3 TenantUsecase

核心方法：

- **Create**：校验 slug 唯一 → repo.Create → AddMember(owner, active) → WriteTuples(user→tenant owner)；FGA 失败回滚 RemoveMember + Delete
- **CreateWithDefaults**：Create → OrganizationUsecase.CreateDefault(tenantID) → ProjectUsecase.CreateDefault(orgID)（一体化流程）
- **EnsurePersonalTenant**：按 userID 查 GetPersonalTenantByUserID → 不存在则 CreateWithDefaults(kind=personal)
- **AddMember/RemoveMember/UpdateMemberRole**：先 DB 后 FGA，FGA 失败回滚 DB（与现有 org/project 模式一致）
- **InviteMember**：AddMember(status=invited) → WriteTuples；被邀请人尚无权限，FGA tuple 预写但 status 控制实际访问
- **AcceptInvitation**：UpdateMemberStatus(active) + 更新 joined_at
- **Delete**：软删除，保留数据

### 3.4 OrganizationMember / ProjectMember 邀请方法

在 OrganizationUsecase 和 ProjectUsecase 中新增：

- **InviteMember**：AddMember(status=invited) → WriteTuples
- **AcceptInvitation**：UpdateMemberStatus(active) + 更新 joined_at

---

## 4. 去除 TenantRootID 硬编码

### 4.1 现状

- `biz.TenantRootID` 类型，Wire 注入单个 ID
- `OrganizationUsecase`、`UserUsecase` 持有固定 `tenantID string`
- `data/seed.go` 的 `NewTenantRootID` 查 slug=root 返回固定 ID
- Authz 中间件 `WithTenantRootID` 用于 `AUTHZ_MODE_OBJECT` + `tenant` 类型鉴权

### 4.2 改造

1. **OrganizationUsecase**：去除 `tenantID string` 字段；Create/CreateDefault 的 tenantID 改为从参数传入（service 层从 actor scope 获取）
2. **UserUsecase**：去除 `tenantID string` 字段；collectUserFGATuples 和 CompensateUserPurge 改为依赖 TenantRepo.ListMembershipsByUserID 获取用户所属的所有 tenant
3. **Authz 中间件**：`AUTHZ_MODE_OBJECT` + `tenant` 类型改为从请求中提取 `tenant_id` 字段；去除 `WithTenantRootID` 选项
4. **Seed**：保留 seedTenant 创建初始 tenant（slug=default），同时 seed `platform:default` admin tuple。不再作为全局 TenantRootID 注入

---

## 5. Tenant Scope 机制

### 5.1 X-Tenant-ID Header

与现有 X-Organization-ID 模式一致，在 `pkg/actor` 中扩展：

- 新增 `TenantID` 字段到 actor.Actor 结构体
- Middleware 从 `X-Tenant-ID` header 解析并注入 actor scope
- Service 层通过 `actor.FromContext(ctx).TenantID` 获取当前 tenant

### 5.2 Organization 按 tenant_id 过滤

- `OrganizationRepo` 的 `ListByUserID` 增加 `tenantID string` 参数
- data 层在 `tenantID != ""` 时加 `Where(organization.TenantIDEQ(tid))`
- 与 project/application 的 orgID 过滤模式一致（defense-in-depth）

---

## 6. Personal Tenant

### 6.1 生命周期

```
用户注册/首次登录
  │
  ├──▶ EnsurePersonalTenant(userID)
  │      ├── 查 GetPersonalTenantByUserID → 已存在则返回
  │      └── 不存在 → CreateWithDefaults(kind=personal, name="{userName}'s Space")
  │            ├── 创建 Tenant(kind=personal)
  │            ├── 创建默认 Organization
  │            ├── 创建默认 Project
  │            └── 用户为 owner
  │
  └──▶ 返回 personal tenant ID
```

### 6.2 约束

- 每用户只有一个 personal tenant（unique constraint 或查询保证）
- Personal tenant 不允许邀请其他成员
- Personal tenant 不允许删除（只要用户存在就保留）

---

## 7. 邀请流程

### 7.1 状态机

```
                            accept
          ┌──────────┐ ──────────▶ ┌──────────┐
          │ invited  │              │  active  │
          └──────────┘ ◀────────── └──────────┘
                            (不支持回退)
               │
               │ reject / expire
               ▼
          (删除记录)
```

### 7.2 FGA 与 status 的关系

采用「预写 FGA tuple」策略：
- InviteMember 时同时写 DB 记录（status=invited）和 FGA tuple
- 被邀请人在 status=invited 时已有 FGA 权限（简化实现）
- 如果需要「邀请未接受不给权限」，后续可改为 AcceptInvitation 时才写 FGA tuple

选择预写的理由：与现有 AddMember 流程一致；减少 AcceptInvitation 的失败面。

---

## 8. Service 层与 Proto

### 8.1 Tenant Proto（新增）

- `api/protos/tenant/service/v1/tenant.proto`：Tenant CRUD + Member 管理消息定义
- `app/iam/service/api/protos/iam/service/v1/i_tenant.proto`：IAM 服务的 Tenant 路由

### 8.2 路由

| 方法 | 路径 | 权限 |
|------|------|------|
| POST | `/v1/tenants` | platform:default / can_manage_all_tenants |
| GET | `/v1/tenants` | 列出当前用户所属的 tenant |
| GET | `/v1/tenants/{tenant_id}` | tenant:{id} / can_view |
| PUT | `/v1/tenants/{tenant_id}` | tenant:{id} / can_manage |
| DELETE | `/v1/tenants/{tenant_id}` | tenant:{id} / can_manage |
| POST | `/v1/tenants/{tenant_id}/members/invite` | tenant:{id} / can_manage_members |
| POST | `/v1/tenants/{tenant_id}/members/{user_id}/accept` | 被邀请人自己 |
| DELETE | `/v1/tenants/{tenant_id}/members/{user_id}` | tenant:{id} / can_manage_members |
| GET | `/v1/tenants/{tenant_id}/members` | tenant:{id} / can_view |

### 8.3 Service 实现

- `internal/service/tenant.go`：TenantService，依赖 TenantUsecase

---

## 9. 涉及文件

| 类别 | 文件 | 变更 |
|------|------|------|
| FGA | `manifests/openfga/model/servora.fga` | 新增 platform type、tenant 扩展 owner、organization 增加 from tenant |
| Ent Schema | `internal/data/schema/tenant.go` | 扩展字段（kind/domain/status/updated_at/SoftDeleteMixin） |
| Ent Schema | `internal/data/schema/tenant_member.go` | 新建 |
| Ent Schema | `internal/data/schema/organization_member.go` | 增加 status 字段 |
| Ent Schema | `internal/data/schema/project_member.go` | 增加 status 字段 |
| Ent Schema | `internal/data/schema/user.go` | 增加 tenant_members edge |
| Entity | `internal/biz/entity/tenant.go` | 新建 Tenant + TenantMember |
| Entity | `internal/biz/entity/organization.go` | OrganizationMember 增加 Status |
| Entity | `internal/biz/entity/project.go` | ProjectMember 增加 Status |
| Biz | `internal/biz/tenant.go` | 新建 TenantUsecase + TenantRepo 接口 |
| Biz | `internal/biz/organization.go` | 去除 tenantID 字段、Create 参数化、InviteMember/AcceptInvitation |
| Biz | `internal/biz/project.go` | InviteMember/AcceptInvitation |
| Biz | `internal/biz/user.go` | 去除 tenantID 字段、适配多 tenant |
| Data | `internal/data/tenant.go` | 新建 TenantRepo 实现 |
| Data | `internal/data/organization.go` | ListByUserID 加 tenantID 过滤 |
| Service | `internal/service/tenant.go` | 新建 TenantService |
| Proto | `api/protos/tenant/service/v1/tenant.proto` | 新建 |
| Proto | `app/iam/service/api/protos/iam/service/v1/i_tenant.proto` | 新建 |
| Middleware | `internal/server/middleware/authz.go` | 去除 WithTenantRootID |
| Actor | `pkg/actor/actor.go` | 增加 TenantID 字段 + X-Tenant-ID 解析 |
| Seed | `internal/data/seed.go` | seed platform:default admin；调整 tenant seed |
| Server | `internal/server/grpc.go`、`http.go` | 注册 TenantService |
| Wire | `cmd/server/wire.go` | 新增 Tenant Provider、去除 TenantRootID |
| Ent Gen | `internal/data/ent/` | make gen.ent 重新生成 |

---

## 参考

- Kemate：`data/schema/tenant.go`、`biz/tenant.go`、`biz/workspace.go`、`openfga/tuples.go`、`service/platform.go`
- go-wind-admin：`data/ent/schema/tenant.go`、`role.go`（扁平 Casbin + 角色模板）
- Casdoor：层级角色可选（Casbin g2）、单 org 限制（issue #4156）
- servora：`docs/design/multi-tenancy-prerequisites.md`
