# 任务：数据层 Scope 与 OpenFGA 一致性

## 第一阶段：数据层 Scope 盘点与约定

- [x] **T1: 盘点需按 org/project 过滤的 repo 方法**
  - 遍历 `internal/data/organization.go`、`project.go`、`application.go`、OrganizationMember、ProjectMember 相关
  - 列出 List/Query/Delete 等方法，标记哪些必须带 organizationID 或 projectID；标注当前是否已带 Where
  - 输出：清单（可写在 design 或单独 doc）

- [x] **T2: 补齐遗漏的 scope 过滤**
  - 根据 T1 清单，对未带 scope 的查询补上 OrganizationIDEQ(orgID) / ProjectIDEQ(projectID) 等
  - 确保调用链：Service 从 requireOrgScope/Actor 取 scope → Biz → Data 显式 Where

- [x] **T3: 可选 — data 层 scope predicate 辅助**
  - 评估后决定跳过：内联 `uuid.Parse(orgID)` 模式清晰直接，引入泛型 helper 反增复杂度

## 第二阶段：OpenFGA 创建/成员变更回滚

- [x] **T4: Organization Create 顺序与回滚**
  - 文件：`internal/biz/organization.go`
  - 顺序：Create → AddMember(owner) → WriteTuples
  - FGA 或 AddMember 失败：回滚（RemoveMember、Delete organization）
  - 与 Kemate biz/tenant.go 模式一致

- [x] **T5: Project Create 顺序与回滚**
  - 文件：`internal/biz/project.go`
  - 顺序：Create → AddMember(admin) → WriteTuples
  - FGA 或 AddMember 失败：回滚（RemoveMember、Delete project）

- [x] **T6: AddMember / RemoveMember / UpdateMemberRole 回滚**
  - Organization 与 Project 的 AddMember：先 DB AddMember，再 WriteTuples；失败则 RemoveMember 回滚
  - RemoveMember / UpdateMemberRole：先 DB 后 FGA；FGA 失败则回滚 DB 变更

## 第三阶段：Ent 查询规范文档

- [x] **T7: 成文 Ent 查询规范**
  - 文件：`docs/development/ent-query-scope.md` 或并入现有开发文档
  - 内容：凡带 organization_id/project_id 的实体，List/Query 必须带 scope 且在 Where 中使用；禁止裸查
  - 注明 Code Review 检查项

- [x] **T8: 验证**
  - `go build ./app/iam/service/...`；跑现有单测；可选手工验证创建/成员变更失败时 DB 回滚正确
