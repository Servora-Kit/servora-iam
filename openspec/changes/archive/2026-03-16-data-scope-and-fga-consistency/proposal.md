# 多租户前置：数据层 Scope 与 OpenFGA 一致性

## 动机

Actor Scope 注入已完成（请求级 X-Organization-ID / X-Project-ID → Actor → Authz + Service）。多租户数据隔离与权限一致还需补齐：

1. **数据层按 scope 过滤**：所有按组织/项目范围的 List/Query 必须显式带 `organization_id` / `project_id` 条件，避免跨租户泄露；当前部分接口已做，未系统盘点与约定。
2. **OpenFGA 创建回滚**：创建 Organization/Project 或 AddMember 时，若 FGA 写入失败，应回滚已执行的 DB 步骤，避免「DB 有记录、FGA 无 tuple」的不一致。
3. **租户/组织生命周期**：在现有创建流程上保证「DB → Member → FGA」顺序及回滚即可；不在本提案引入「创建租户」API。
4. **Ent 查询规范**：成文约定 + Code Review 检查项，确保新增/修改 repo 时凡涉及 org/project 范围必须带 Where；可选 data 层 predicate 辅助。

参照 Kemate（data 显式 Where、biz 创建顺序与 FGA 失败回滚）与 docs/design/multi-tenancy-prerequisites.md。

## 目标

1. 盘点并约定：所有需按 org/project 过滤的 repo 方法必须接收 scope 并在 Where 中使用。
2. 将 Organization/Project 创建与 AddMember/RemoveMember/UpdateRole 的流程改为「先 DB 后 FGA，FGA 失败则回滚 DB」。
3. 文档化 Ent 查询规范，并可选提供 data 层 scope predicate 辅助。
4. 不引入 Ent Privacy、不新增「创建租户」API。

## Capabilities

- **data-scope-audit**：数据层 scope 盘点与约定
- **fga-create-rollback**：创建/成员变更时 FGA 失败回滚 DB
- **ent-query-convention**：Ent 按 org/project 查询的成文约定与 CR 检查

## 非目标

- 不引入 Ent Privacy Policy 自动过滤（后续可选）。
- 不新增「创建租户」API（保持单租户根或后续单独提案）。
- PurgeUser 级联删除由独立提案 purge-user-cascade 覆盖。
