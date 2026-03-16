# 设计：数据层 Scope 与 OpenFGA 一致性

## 1. 数据层按 scope 过滤

### 约定

- 凡「按组织/项目范围」的接口，Service 层必须调用 `requireOrgScope(ctx)` 或从 Actor 取 projectID，将 orgID/projectID 传入 Biz → Data。
- Data 层**必须**在查询中加 `OrganizationIDEQ(orgID)` / `ProjectIDEQ(projectID)`，不允许裸查。
- 参照 Kemate：repo 内显式 Where，无「从 context 自动注入」的 data 层 helper。

### 盘点范围

- `internal/data/organization.go`：List、ListByUser、PurgeCascade 等是否都带 scope 或按 ID。
- `internal/data/project.go`：List、Delete 等已用 `OrganizationIDEQ(oid)`，确认无遗漏。
- `internal/data/application.go`：List 已用 `OrganizationIDEQ`，确认无遗漏。
- `internal/data/*member*`：OrganizationMember、ProjectMember 的 List/Delete 是否都带 orgID/projectID 或 userID，且不裸查跨租户。

### 可选

- 在 data 层提供 `scope.go`，例如 `OrgScope(orgID)` 返回 `project.OrganizationIDEQ(orgID)` 等 predicate，供多处复用，**不**做从 context 自动取 scope。

---

## 2. OpenFGA 创建与成员变更回滚

### 顺序与回滚策略（参照 Kemate）

- **Organization Create**：DB Create → AddMember(owner) → WriteTuples(tenant→org, user→owner)；任一步失败则回滚（RemoveMember、Delete org）。
- **Project Create**：DB Create → AddMember(admin) → WriteTuples(org→project, user→admin)；任一步失败则回滚（RemoveMember、Delete project）。
- **AddMember（org/project）**：DB AddMember → WriteTuples；失败则 DB RemoveMember 回滚。
- **RemoveMember / UpdateMemberRole**：先 DB 后 FGA（DeleteTuples 或 Delete+Write）；FGA 失败则回滚 DB。

### 涉及文件

- `internal/biz/organization.go`：Create、CreateDefault、AddMember、RemoveMember、UpdateMemberRole
- `internal/biz/project.go`：Create、CreateDefault、AddMember、RemoveMember、UpdateMemberRole
- 当前为「先 DB 再 FGA 再 AddMember」且 FGA 失败仅 `_ = uc.authz.WriteTuples`；需改为 FGA 失败时执行已做步骤的回滚（RemoveMember、Delete 等）。

### 可借鉴

- Kemate：`biz/tenant.go` Create、`biz/workspace.go` Create/AddMember 的回滚顺序与代码结构。

---

## 3. 租户/组织生命周期（本提案范围）

- 不新增「创建租户」API；仅保证现有「创建组织 → 创建项目 → 添加成员」与 FGA 一致且在失败时回滚（见上节）。

---

## 4. Ent 查询规范

- 在 `docs/development/` 或 `docs/design/` 下成文：「凡查询 OrganizationMember、Project、Application、ProjectMember 等带 org/project 的实体，必须传入 scope（orgID/projectID）并在 Where 中使用；禁止在未带 scope 的 path 下做全表 List。」
- Code Review 检查项：新增/修改 repo 的 List/Query 时，确认是否涉及 org/project 范围，若是则必须有对应 Where。

---

## 参考

- docs/design/multi-tenancy-prerequisites.md：§1～§4
- Kemate：data/workspace.go、application.go；biz/tenant.go、workspace.go；service/scope.go
