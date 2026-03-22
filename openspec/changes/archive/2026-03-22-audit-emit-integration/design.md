## Context

Phase 1（framework-audit-skeleton）交付了审计事件模型（`pkg/audit`）和消息代理（`pkg/broker`），但事件的**产出端**尚未接线。当前 `pkg/authz` 的 Check 和 `pkg/openfga` 的 tuple 操作执行后没有任何审计输出。

同时，`pkg/openfga` 在早期 IAM 开发中引入了业务特化代码：`"user:"` 前缀硬编码、IAM model computed relations 写死在缓存逻辑中。这些在框架化之前必须清理。

本次变更涉及三个 `pkg/` 包和一个 `app/` 服务的适配。

```
pkg/openfga  ──(框架化 + audit)──▶  pkg/audit.Recorder
pkg/authz    ──(audit)──────────▶  pkg/audit.Recorder
app/iam/service  ──(适配 API 变更)──▶  pkg/openfga (new API)
```

## Goals / Non-Goals

**Goals:**
- `pkg/openfga` API 去特化，支持任意 principal 类型
- `pkg/openfga` 引入 ClientOption 模式，注入 AuditRecorder 和 ComputedRelationMap
- `pkg/openfga` tuple 操作拆 core/public 分层，成功后自动 emit audit 事件
- `pkg/openfga` CachedCheck 返回 cacheHit 信息
- `pkg/authz` Check 后自动 emit authz.decision 审计事件
- 端到端验证：LogEmitter + BrokerEmitter 两种模式

**Non-Goals:**
- 不新建 Audit Service（Phase 2b）
- 不改 `pkg/audit` Recorder/Emitter 核心接口
- 不改 audit.proto / annotations.proto
- 不实现 authn.result 或 resource.mutation 事件

## Decisions

### D1: pkg/openfga API 去特化

**决策：** `Check`/`ListObjects`/`CachedCheck`/`CachedListObjects` 的 `userID string` 参数改为 `user string`，调用方传完整 OpenFGA principal（如 `"user:uuid"`、`"service:gateway"`）。

**替代方案：** 保留 `userID` 并增加 `userType` 参数 → 拒绝：增加参数数量，且不符合 OpenFGA SDK 的 `User` 字段语义（本身就是 `type:id` 格式）。

**迁移影响：** 所有调用方需从 `Check(ctx, userID, ...)` 改为 `Check(ctx, "user:"+userID, ...)`。当前调用方：`pkg/authz`（1 处）、`app/iam/service`（多处）。

### D2: pkg/openfga 缓存层去特化

**决策：**
- `parseTupleComponents` 改为通用 `type:id` 解析，不再限定 `"user:"` 前缀
- `affectedRelations` 中的硬编码 `computedByType` map 移除，改为 `Client` 持有的 `ComputedRelationMap map[string][]string`，通过 `WithComputedRelations` option 注入
- 默认 map 为空：只失效 tuple 自身 relation
- IAM 服务在 `NewClient` 时传入自己的 model 映射

**替代方案：** 将缓存逻辑完全移出 `pkg/openfga` → 拒绝：缓存是框架级通用能力，只是 computed relation 映射是业务特定的。

### D3: pkg/openfga ClientOption 模式

**决策：** `NewClient(cfg *conf.App_OpenFGA, opts ...ClientOption) (*Client, error)`

```go
type ClientOption func(*clientOptions)

type clientOptions struct {
    recorder           *audit.Recorder
    computedRelations  map[string][]string
}
```

Options:
- `WithAuditRecorder(r *audit.Recorder)` — 注入审计 recorder（nil-safe）
- `WithComputedRelations(m map[string][]string)` — 注入缓存失效用的 computed relation 映射

### D4: pkg/openfga core/public 分层

**决策：** `WriteTuples`/`DeleteTuples` 拆为：
- `writeTuplesCore` / `deleteTuplesCore` — 纯 SDK 操作（unexported）
- `WriteTuples` / `DeleteTuples` — public wrapper，组合 core + audit emit

```
WriteTuples(ctx, tuples...)
  ├── writeTuplesCore(ctx, tuples...)     // 纯 SDK 调用
  └── if err == nil && c.recorder != nil  // cross-cutting
      └── recorder.RecordTupleChange(...)
```

`EnsureTuples` 内部调用 `WriteTuples`（已含 audit），无需额外处理。

### D5: CachedCheck 返回值扩展

**决策：** `CachedCheck(...) (allowed bool, cacheHit bool, err error)`

- 命中缓存时 `cacheHit = true`
- 未命中（实际调用 OpenFGA Check 后写入缓存）时 `cacheHit = false`
- `rdb == nil` 降级为 plain Check 时 `cacheHit = false`

调用方仅 `pkg/authz` middleware 一处，直接适配。

### D6: pkg/authz audit 集成

**决策：** `authzConfig` 新增 `recorder *audit.Recorder` 字段，通过 `WithAuditRecorder` option 注入。

Check 完成后调用：
```go
if cfg.recorder != nil {
    cfg.recorder.RecordAuthzDecision(ctx, operation, a, audit.AuthzDetail{
        Relation:    relation,
        ObjectType:  objectType,
        ObjectID:    objectID,
        Decision:    decision, // allowed / denied / error
        CacheHit:   cacheHit,
        ErrorReason: errMsg,
    })
}
```

需适配 openfga API 变更：传完整 principal `"user:" + a.ID()` 而非裸 `a.ID()`。

### D7: app/iam/service 适配

**决策：** IAM 服务所有 `pkg/openfga` 调用点适配：
- `Check` / `CachedCheck` / `ListObjects` / `CachedListObjects`：`userID` → `"user:" + userID`
- `InvalidateCheck` / `InvalidateListObjects` / `InvalidateForTuples`：同步适配
- `NewClient` 调用传入 `WithComputedRelations(iamComputedRelations)`

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| Breaking change 范围：`pkg/openfga` 签名变更影响所有调用方 | 调用方有限（authz 1 处 + iam 多处），且均在同一 repo。一次性机械替换 |
| `CachedCheck` 三返回值使调用方稍复杂 | 只有 authz middleware 一处调用，且新增的 `cacheHit` 直接传给 audit |
| audit emit 增加每次 Check/Write 的延迟 | Emitter.Emit 异步（BrokerEmitter 内部 Kafka produce 是异步的）；LogEmitter 仅 JSON marshal + log write，微秒级 |
| IAM 服务当前从工具链移除，适配后测试困难 | `pkg/openfga` 和 `pkg/authz` 有独立单元测试；IAM 适配为机械改动，编译通过即可 |
