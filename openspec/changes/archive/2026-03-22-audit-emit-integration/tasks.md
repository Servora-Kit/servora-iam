## 1. pkg/openfga ClientOption 模式引入

- [x] 1.1 在 `pkg/openfga/client.go` 新增 `ClientOption` 类型、`clientOptions` struct、`WithAuditRecorder`、`WithComputedRelations` option 函数
- [x] 1.2 修改 `Client` struct，新增 `recorder *audit.Recorder` 和 `computedRelations map[string][]string` 字段
- [x] 1.3 修改 `NewClient` 签名为 `NewClient(cfg *conf.App_OpenFGA, opts ...ClientOption) (*Client, error)`，应用 options
- [x] 1.4 修改 `NewClientOptional` 透传 options：`NewClientOptional(cfg *conf.App, l logger.Logger, opts ...ClientOption) *Client`
- [x] 1.5 编写 `NewClient` option 注入的单元测试

## 2. pkg/openfga API 去特化

- [x] 2.1 修改 `Check(ctx, userID, ...)` 为 `Check(ctx, user, ...)`，移除内部 `"user:" + userID` 拼接
- [x] 2.2 修改 `ListObjects(ctx, userID, ...)` 为 `ListObjects(ctx, user, ...)`，移除内部 `"user:" + userID` 拼接
- [x] 2.3 修改 `CachedCheck` 签名：参数 `userID` → `user`，返回值 `(bool, error)` → `(bool, bool, error)`，增加 cacheHit 逻辑
- [x] 2.4 修改 `CachedListObjects` 签名：参数 `userID` → `user`
- [x] 2.5 修改 `InvalidateCheck` / `InvalidateListObjects` 签名：参数 `userID` → `user`
- [x] 2.6 编写 `Check`/`CachedCheck` 去特化后的单元测试（验证不拼接前缀、cacheHit 返回值）

## 3. pkg/openfga 缓存层去特化

- [x] 3.1 修改 `parseTupleComponents`：移除 `strings.HasPrefix(t.User, "user:")` 限制，改为通用 `type:id` 解析
- [x] 3.2 修改 `affectedRelations`：移除硬编码 `computedByType` map，改为使用 `Client.computedRelations`
- [x] 3.3 将 `InvalidateForTuples` 从 package-level 函数改为 `Client` 方法（需要访问 `computedRelations`）
- [x] 3.4 编写缓存层去特化的单元测试（通用 principal 解析、自定义 computed relations、空 map 默认行为）

## 4. pkg/openfga tuple 操作 core/public 分层 + audit emit

- [x] 4.1 将 `WriteTuples` 当前逻辑提取为 `writeTuplesCore`（unexported），`WriteTuples` 改为 wrapper
- [x] 4.2 将 `DeleteTuples` 当前逻辑提取为 `deleteTuplesCore`（unexported），`DeleteTuples` 改为 wrapper
- [x] 4.3 在 `WriteTuples` wrapper 中添加 audit emit：成功后调用 `recorder.RecordTupleChange`（nil-safe）
- [x] 4.4 在 `DeleteTuples` wrapper 中添加 audit emit：成功后调用 `recorder.RecordTupleChange`（nil-safe）
- [x] 4.5 编写 tuple audit emit 的单元测试（成功写入 emit、失败不 emit、nil recorder 不 emit）

## 5. pkg/authz 适配 + audit 集成

- [x] 5.1 修改 `authzConfig`，新增 `recorder *audit.Recorder` 字段
- [x] 5.2 新增 `WithAuditRecorder(r *audit.Recorder) Option`
- [x] 5.3 适配 `openfga.CachedCheck` API 变更：传完整 principal `"user:" + a.ID()`，接收三返回值 `(allowed, cacheHit, err)`
- [x] 5.4 在 Check 完成后添加 `recorder.RecordAuthzDecision` 调用（allowed / denied / error 三种 decision）
- [x] 5.5 编写 authz audit emit 的单元测试（allowed emit、denied emit、error emit、nil recorder 跳过）
- [x] 5.6 更新 `pkg/authz` 已有测试以适配 openfga API 变更

## 6. app/iam/service 适配

- [x] 6.1 全局搜索 `app/iam/service` 中所有 `openfga.Check` / `openfga.CachedCheck` / `openfga.ListObjects` / `openfga.CachedListObjects` 调用，将 `userID` 参数改为 `"user:" + userID`
- [x] 6.2 全局搜索 `InvalidateCheck` / `InvalidateListObjects` / `InvalidateForTuples` 调用，适配新签名
- [x] 6.3 在 IAM 服务的 `NewClient` 调用处传入 `openfga.WithComputedRelations(iamComputedRelations)`，将原 `affectedRelations` 中的 map 迁移为 IAM 服务本地常量
- [x] 6.4 适配 `CachedCheck` 三返回值（忽略 cacheHit 或按需使用）
- [x] 6.5 确认 `app/iam/service` 编译通过

## 7. 验证与收尾

- [x] 7.1 运行 `go test ./pkg/openfga/...`，确认所有测试通过
- [x] 7.2 运行 `go test ./pkg/authz/...`，确认所有测试通过
- [x] 7.3 运行 `go test ./pkg/audit/...`，确认已有测试不受影响
- [x] 7.4 运行 `make lint.go`，确认无 lint 错误（已有的28个 errcheck + 2个 staticcheck 均为 pre-existing，本次变更文件无新增 lint 问题）
- [x] 7.5 使用 LogEmitter 端到端验证：配置 audit emitter 为 log 模式，发起带 authz 的请求，确认日志中出现 authz.decision 事件 JSON
- [x] 7.6 使用 BrokerEmitter 端到端验证：启动 Kafka（`make compose.up`），配置 audit emitter 为 broker 模式，发起请求，确认 Kafka topic 中可消费到事件
- [x] 7.7 更新 `pkg/openfga/AGENTS.md` 反映 API 变更和 ClientOption 模式
- [x] 7.8 更新 `pkg/authz/AGENTS.md`（如存在）反映 audit 集成
