## Why

Phase 1（framework-audit-skeleton）已交付审计骨架（`pkg/audit` Recorder/Emitter）、消息代理（`pkg/broker`）和 Actor v2，但审计事件的**产出端**尚未接入——`pkg/authz` 每次 Check 不产出 `authz.decision` 事件，`pkg/openfga` 的 tuple 写删不产出 `tuple.changed` 事件。此外，`pkg/openfga` 存在框架化缺陷（硬编码 `"user:"` 前缀、业务 model 写死在缓存逻辑中），需在接入审计前一并修正。

对应主设计文档 **Phase 2a**（`docs/plans/2026-03-20-keycloak-openfga-audit-design.md`）。

## What Changes

- **BREAKING** `pkg/openfga` API 框架化：`Check`/`ListObjects`/`CachedCheck`/`CachedListObjects` 参数从 `userID string` 改为 `user string`（完整 principal），移除 `"user:"` 硬编码
- **BREAKING** `pkg/openfga` `CachedCheck` 返回值扩展为 `(allowed bool, cacheHit bool, err error)`
- `pkg/openfga` 引入 `ClientOption` 模式（`NewClient(cfg, ...ClientOption)`），新增 `WithAuditRecorder`、`WithComputedRelations`
- `pkg/openfga` 缓存层框架化：`parseTupleComponents` 通用化、`affectedRelations` 改为可配置 `ComputedRelationMap`
- `pkg/openfga` `WriteTuples`/`DeleteTuples`/`EnsureTuples` 拆 core + public wrapper，成功后自动 emit `tuple.changed` 事件
- `pkg/authz` 适配 openfga API 变更（传完整 principal）
- `pkg/authz` 新增 `WithAuditRecorder` option，Check 后自动产出 `authz.decision` 事件（含 CacheHit）
- `app/iam/service` 适配 openfga API 变更

## Non-goals

- 不新建 Audit Service 微服务（Phase 2b 范围）
- 不实现 `authn.result` 事件接入（Phase 3 随 Keycloak 改造）
- 不实现 `resource.mutation` 自动化（Phase 4 代码生成）
- 不实现新的 proto 定义（Phase 1 已完成 audit.proto/annotations.proto）
- 不修改 `pkg/audit` Recorder/Emitter 核心接口（已就绪）

## Capabilities

### New Capabilities
- `openfga-framework-api`: pkg/openfga API 去特化——移除硬编码 principal 前缀，缓存层通用化，引入 ClientOption 模式
- `authz-audit-emit`: pkg/authz 审计事件自动产出——Check 后通过 Recorder 发出 authz.decision 事件
- `openfga-audit-emit`: pkg/openfga tuple 审计事件自动产出——WriteTuples/DeleteTuples 后通过 Recorder 发出 tuple.changed 事件

### Modified Capabilities


## Impact

- **pkg/openfga**：Check/ListObjects/CachedCheck/CachedListObjects 签名变更（breaking），所有调用方需适配
- **pkg/authz**：适配 openfga API 变更 + 新增 audit 依赖
- **app/iam/service**：所有 openfga 调用需传完整 principal（如 `"user:" + userID`），缓存失效调用需传完整 principal
- **go.mod**（根模块）：`pkg/openfga` 新增对 `pkg/audit` 和 `pkg/actor` 的依赖
