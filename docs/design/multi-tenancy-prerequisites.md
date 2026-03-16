# 多租户实现前必要前置（细化方案）

本文档参照 **Kemate** 与 **go-wind-admin** 的实现，细化多租户前建议补齐的前置项，并标注可借鉴的现成代码位置。servora 已完成 **Actor Scope 注入**（请求级 org/project 上下文），以下为在此基础上仍需系统化补齐的前置。**第 6 项**与 **TODO.md** 中「级联删除原子性与僵尸数据」一致，以 TODO.md 为准。

---

## 1. 数据层按 scope 过滤

### 目标

所有 List/Query 在 Biz/Data 层**显式**按 `organization_id` / `project_id` 过滤，避免跨租户数据泄露。scope 来源统一为 `actor.OrganizationIDFromContext(ctx)` / `actor.ProjectIDFromContext(ctx)`（已由 Scope 中间件注入）。

### 参照结论

| 项目 | 做法 | 可借鉴点 |
|------|------|----------|
| **Kemate** | Repo 内显式 `Where(workspace.TenantIDEQ)`, `application.WorkspaceIDEQ` 等；**无** data 层“从 context 自动注入”的 helper。Service 层用 `requireWorkspaceScope(ctx)` 取 `(userID, workspaceID)` 再传给 biz/repo。 | 与 servora 的 `requireOrgScope(ctx)` 一致；repo 显式 Where 可照抄模式。 |
| **go-wind-admin** | 无全局自动过滤。Create/Update 用 `SetNillableTenantID(req.Data.TenantId)`；List/Delete 在需要处手写 `Where(role.TenantIDEQ(tenantID))`。tenant 来自 **请求 DTO**，由 service 从 `auth.FromContext` 的 operator 填入。 | servora 已用 Actor scope，不需 operator 填 req；只需统一「scope 从 ctx 取 → 传给 data」+ data 显式 Where。 |

### servora 现状

- **已有**：`internal/data/project.go` 的 List 用 `project.OrganizationIDEQ(oid)`，`application.go` 的 List 用 `application.OrganizationIDEQ(uid)`；oid/uid 来自 service 层（当前即 `requireOrgScope` 的 orgID）。
- **缺口**：未系统化盘点所有「应按 org/project 过滤」的查询；无统一约定「凡带 org 的 repo 方法必须接收 orgID 参数并在 Where 中使用」。

### 可借鉴代码（Kemate）

- `app/kemate/service/internal/data/workspace.go`：`Where(workspace.TenantIDEQ(tenantID))`、`Where(application.WorkspaceIDEQ(workspaceID))`。
- `app/kemate/service/internal/data/application.go`：`application.WorkspaceIDEQ(wid)`，wid 由 service 传入。
- `app/kemate/service/internal/service/scope.go`：`requireWorkspaceScope(ctx)` 返回 `(userID, workspaceID)`，与 servora 的 `requireOrgScope` 同模式。

### 细化任务

1. **盘点**：列出所有 repo 的 List/Query/Delete 方法，标记哪些必须带 `organizationID` 或 `projectID` 过滤（参考 `internal/data/organization.go`、`project.go`、`application.go`、member 相关）。
2. **约定**：凡「按组织/项目范围」的接口，Service 层必须调用 `requireOrgScope(ctx)` 或从 Actor 取 projectID，将 orgID/projectID 传入 Biz → Data，Data 层**必须**在查询中加 `OrganizationIDEQ(orgID)` / `ProjectIDEQ(projectID)`，不允许裸查。
3. **可选**：在 data 层提供 `ScopeByOrg(orgID)` 返回 `func(q *ent.ProjectQuery)` 的 predicate 组合，仅用于统一写法，不改变「显式传 scope」的约定（Kemate/go-wind-admin 均无自动注入，保持显式更稳妥）。

---

## 2. OpenFGA 模型与数据

### 目标

确认 FGA model 中 tenant → organization → project → member 关系完整；创建/变更组织、项目、成员时，**可靠写入** OpenFGA tuple，并在失败时有明确处理（回滚或补偿）。

### 参照结论

| 项目 | 做法 | 可借鉴点 |
|------|------|----------|
| **Kemate** | 模型在 `manifests/openfga/model/kemate.fga`；写入在 `app/kemate/service/internal/openfga/tuples.go`（`WriteWorkspaceRole`、`WriteWorkspaceTenant`、`WriteTenantRole`、`WriteResourceWorkspace`）。Create 流程：DB → AddMember → FGA；**FGA 失败则回滚 DB**（RemoveMember + Delete）。 | 顺序与回滚策略可直接参考。 |
| **go-wind-admin** | 未使用 OpenFGA；用 Casbin + Domain=tenant_id 做 RBAC。 | 无可借鉴的 FGA 部分。 |

### servora 现状

- **模型**：`manifests/openfga/model/servora.fga` 已定义 `tenant`、`organization`、`project`、`user` 及 can_view/can_manage 等关系，结构完整。
- **写入**：`internal/biz/organization.go`、`project.go` 在 Create/AddMember/RemoveMember/UpdateRole 等处调 `uc.authz.WriteTuples`；tuple 格式与 model 一致（tenant→organization、organization→project、user→org/project 角色）。
- **缺口**：Create 顺序为「先 DB 再 FGA 再 AddMember」，且 FGA/AddMember 失败仅打日志（`_ = uc.authz.WriteTuples`、`Warnf`），**无回滚**，易产生「DB 有记录、FGA 无 tuple」的不一致。

### 可借鉴代码（Kemate）

- `app/kemate/service/internal/biz/tenant.go`（Create）：`repo.Create` → `repo.AddMember` → `authz.WriteTenantRole`；失败则 `RemoveMember` + `repo.Delete`。
- `app/kemate/service/internal/biz/workspace.go`（Create）：`repo.Create` → `repo.AddMember` → `authz.WriteWorkspaceRole` → `authz.WriteWorkspaceTenant`；任一步失败则回滚前面步骤（DeleteWorkspaceRole、RemoveMember、Delete）。
- `app/kemate/service/internal/biz/workspace.go`（AddMember）：`repo.AddMember` → `authz.WriteWorkspaceRole`；失败则 `repo.RemoveMember`。
- `app/kemate/service/internal/openfga/tuples.go`：具体 Tuple 构造与 API 调用方式（servora 已有 pkg/openfga 与 data/authz，可对照 tuple 形状与错误处理）。

### 细化任务

1. **顺序与回滚**：将「创建组织/项目」改为「DB Create → AddMember → WriteTuples」，任一步失败则回滚已执行步骤（参考 Kemate 的 RemoveMember + Delete + DeleteWorkspaceRole 等）；至少 FGA 失败时**必须**回滚 DB 或提供补偿脚本。
2. **AddMember/RemoveMember/UpdateRole**：同样约定「先 DB 后 FGA，FGA 失败则回滚 DB 变更」。
3. **初始化/迁移**：确认 `make openfga.init` / seed 在无 FGA 时不影响启动；若有存量数据，提供「按 tenant/org 同步 FGA」的脚本或管理接口（Kemate 无现成脚本，可自行设计）。

---

## 3. 租户/组织生命周期

### 目标

租户(Tenant)、组织(Organization)、项目(Project) 的创建与成员加入/离开流程完整且与 OpenFGA 一致；多租户下「用户属于多组织、组织属于单租户」的模型清晰。

### 参照结论

| 项目 | 做法 | 可借鉴点 |
|------|------|----------|
| **Kemate** | Tenant 创建：PlatformService.CreateTenant → biz CreateTenant（Create + AddMember + WriteTenantRole）。Workspace 创建：需要 tenant 上下文，Create + AddMember + WriteWorkspaceRole + WriteWorkspaceTenant。 | 创建入口、biz 内步骤顺序与 FGA 写入时机。 |
| **go-wind-admin** | TenantService.Create / CreateTenantWithAdminUser（租户 + 管理员用户 + 角色模板复制 + Membership）；OrgUnitService.Create 带 TenantId。 | 租户与「首个管理员」一体创建的流程；servora 若需「租户+默认组织+默认项目」可参考。 |

### servora 现状

- **已有**：Organization 创建（含 tenant_id）、Project 创建（含 organization_id）、AddMember/RemoveMember/UpdateMemberRole；FGA tuple 在 biz 层写入。
- **缺口**：Tenant 本身在 servora 中多为「单租户根 ID」配置（biz.TenantRootID），无「创建租户」的 API；若未来支持多租户根，需补 Tenant 创建与 FGA 的 tenant 写入。组织/项目生命周期已存在，主要补齐「FGA 失败回滚」即可（见上一节）。

### 可借鉴代码（Kemate）

- `app/kemate/service/internal/service/platform.go`：CreateTenant 入口。
- `app/kemate/service/internal/biz/tenant.go`：Create、AddMember 及 WriteTenantRole 与回滚。
- `app/kemate/service/internal/biz/workspace.go`：Create、AddMember、WriteWorkspaceRole、WriteWorkspaceTenant 及回滚。

### 细化任务

1. **当前阶段**：不强制引入「创建租户」API；确保现有「创建组织 → 创建项目 → 添加成员」与 FGA 一致并在失败时回滚。
2. **若引入多租户根**：参考 Kemate 的 CreateTenant + go-wind-admin 的 CreateTenantWithAdminUser，设计「创建租户 + 默认组织 + 默认项目 + 当前用户为 owner」的一体化流程，并同步 FGA。

---

## 4. Ent 查询规范（含可选 Scope 辅助）

### 目标

约定：凡带 `organization_id` / `project_id` 的实体，**所有** List/GetByScope/Delete 等查询必须带对应 Where 条件；避免遗漏导致跨租户可见。可选：在 data 层提供轻量 predicate 辅助，不改变「显式传 scope」的约定。

### 参照结论

| 项目 | 做法 | 可借鉴点 |
|------|------|----------|
| **Kemate** | 无 Ent Privacy、无自动 Where；repo 内统一用 `WorkspaceIDEQ`、`TenantIDEQ` 等显式过滤。 | 保持显式 Where，不引入「从 context 自动加条件」的拦截器。 |
| **go-wind-admin** | Schema 用 `mixin.TenantID[uint32]{}`；查询时手写 `Where(role.TenantIDEQ(tenantID))` 或 `SetNillableTenantID`；无 QueryHook 自动注入。 | Mixin 仅提供字段与索引；过滤仍靠显式 Where，servora 已用 Edge 不必改 schema。 |

### servora 现状

- **已有**：Ent schema 中 Organization、Project、Application、OrganizationMember、ProjectMember 等均有 `organization_id` 或 `project_id` 及 Edge；Where 已在 project/application List 等处使用。
- **缺口**：无成文约定与检查清单；新加 repo 方法时容易遗漏 Where。

### 可借鉴代码（Kemate）

- `app/kemate/service/internal/data/workspace.go`：多处 `Where(workspace.TenantIDEQ(tenantID))`、`Where(application.WorkspaceIDEQ(workspaceID))` 的写法。
- `app/kemate/service/internal/data/application.go`：`application.WorkspaceIDEQ(wid)`。

### 细化任务

1. **文档约定**：在 `docs/development/` 或本目录下写明「凡查询 OrganizationMember、Project、Application、ProjectMember 等带 org/project 的实体，必须传入 scope（orgID/projectID）并在 Where 中使用；禁止在未带 scope 的 path 下做全表 List」。
2. **Code Review 检查项**：新增/修改 repo 的 List/Query 时，必须确认是否涉及 org/project 范围，若是则必须有对应 Where。
3. **可选**：在 `internal/data` 增加 `scope.go`，提供例如 `OrgScope(orgID)` 返回 `project.OrganizationIDEQ(orgID)` 等，供多个 repo 复用同一 predicate 写法，**不**做「从 context 自动取 scope」的注入（与 Kemate/go-wind-admin 一致）。

---

## 5. Ent Privacy（可选）

### 目标

若希望「所有查询自动带 scope」，可引入 Ent Privacy Policy，在 Query/Mutation 规则中根据 context 中的 Actor scope 自动加 Where；否则继续依赖「显式 Where + 约定 + Code Review」（与 Kemate、go-wind-admin 一致）。

### 参照结论

| 项目 | 做法 | 可借鉴点 |
|------|------|----------|
| **Kemate** | 仅有 Ent 生成的 `ent/privacy/privacy.go`，**无**按 workspace/tenant 的自定义 Policy；多租户隔离完全靠 repo 显式 Where。 | 不引入 Privacy 也能安全多租户，靠约定与显式过滤。 |
| **go-wind-admin** | 无 Ent Privacy；用 mixin.TenantID 提供字段，手写 Where。 | 同上。 |

### servora 现状

- **已有**：Ent 未配置 Privacy；当前依赖 service 层 `requireOrgScope` + data 层显式 Where。
- **缺口**：无；若不做 Privacy，保持现状即可。

### 细化任务

1. **短期**：不引入 Ent Privacy；多租户前置依赖上述 1～4 的显式过滤与 FGA 一致性。
2. **若后续引入**：需设计「从 context 取 Actor → 取 OrganizationID/ProjectID → 在对应实体的 Query 规则中注入 predicate」的 Policy，并全面测试避免误伤「跨 org 的管理员」等场景；可参考 Ent 官方文档与 Kemate 的「无 Privacy」做法做对比评估。

---

## 6. PurgeUser 级联删除原子性与僵尸数据（参见 TODO.md）

与 **TODO.md** 中「级联删除原子性与僵尸数据」同一事项，多租户下更需保证用户删除后不留下跨系统脏数据与孤儿行。

### 目标

- **PurgeUser** 跨 FGA / Redis / Postgres 三步时，顺序或补偿策略明确，避免「DB 已删、FGA/Redis 仍留」的不一致。
- **PurgeCascade** 在同一事务内删除该用户作为成员的 OrganizationMember、ProjectMember 以及**该用户拥有的** Organization、Project（或通过 schema 外键 CASCADE 由 DB 级联），避免孤儿行。

### 待办（与 TODO.md 一致）

1. 将 PurgeUser 顺序改为**先 DB（PurgeCascade）再 FGA 再 Redis**，或为 FGA/Redis 做补偿/重试。
2. 在 PurgeCascade 同一事务内按依赖顺序删除该用户拥有的 Organization、Project（或通过 schema 外键 CASCADE 由 DB 级联）。
3. 可选：PurgeUser 打点/日志，便于排查中断点；提供按 `user_id` 清理 FGA/Redis 残留的补偿脚本或管理接口。

### 说明

本节不重复 TODO.md 全文，实现时以 **TODO.md** 为准；此处仅标明其属于「多租户前建议补齐」的一环，与上述 1～5 并列。

---

## 总结表

| 前置项 | 参照项目可借鉴代码 | servora 当前缺口 | 建议动作 |
|--------|--------------------|------------------|----------|
| 1. 数据层按 scope 过滤 | Kemate: data/workspace.go, application.go; service/scope.go | 未系统盘点；无成文约定 | 盘点所有需按 org/project 过滤的 repo；约定 + 可选 predicate 辅助 |
| 2. OpenFGA 模型与数据 | Kemate: biz/tenant.go, workspace.go; openfga/tuples.go | Create 无 FGA 失败回滚 | 调整顺序为 DB→Member→FGA；FGA 失败回滚 DB |
| 3. 租户/组织生命周期 | Kemate: service/platform.go, biz/tenant.go, workspace.go | 无「创建租户」API（可暂不做） | 先保证 org/project/member 与 FGA 一致+回滚 |
| 4. Ent 查询规范 | Kemate: data 显式 Where；go-wind-admin: mixin+Where | 无成文约定与检查项 | 文档约定 + CR 检查 + 可选 scope predicate 辅助 |
| 5. Ent Privacy | 两项目均未使用 | 无 | 短期不做；后续可选评估 |
| 6. PurgeUser 级联删除 | — | 顺序未定；PurgeCascade 未删用户拥有的 Org/Project | 见 TODO.md：先 DB 再 FGA/Redis；Cascade 补删或 DB CASCADE；可选补偿脚本 |

---

## 参考路径速查（Kemate）

- Service 层 scope：`app/kemate/service/internal/service/scope.go`（requireWorkspaceScope）
- AuthZ 写入 viewer：`app/kemate/service/internal/server/middleware/auth/authz.go`（SetWorkspaceID）
- Data 过滤：`app/kemate/service/internal/data/workspace.go`、`application.go`
- Biz 创建与回滚：`app/kemate/service/internal/biz/tenant.go`、`workspace.go`
- FGA 写入：`app/kemate/service/internal/openfga/tuples.go`
- FGA 模型：`manifests/openfga/model/kemate.fga`

## 参考路径速查（go-wind-admin）

- Viewer/Operator：`backend/pkg/middleware/auth/auth.go`、`pkg/entgo/viewer/user_viewer.go`
- Service 用 operator：`auth.FromContext(ctx)` 取 TenantId 等填 req
- Data 层 Where/Set：`backend/app/admin/service/internal/data/role_repo.go`、`role_permission_repo.go`、`org_unit_repo.go`
- 租户创建：`backend/app/admin/service/internal/service/tenant_service.go`（Create、CreateTenantWithAdminUser）
