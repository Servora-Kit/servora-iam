## 上下文

Traefik ForwardAuth 已接入，认证流程变为：

```
Client → Traefik → ForwardAuth (IAM /v1/auth/verify) → 204 + X-User-ID
       → Traefik 转发到目标服务（带 X-User-ID 头）
```

当前 `pkg/transport/server/middleware/` 包含完整的 Authn/Authz 中间件，但只有 IAM 服务使用。其中 `Authz` 中间件定义了自己的 `AuthzRuleEntry` 类型，与 protoc 插件生成的 `iamv1.AuthzRuleEntry` 结构相同但属于不同包，导致每个服务都需要一个 `convertAuthzRules` 桥接函数。

其他服务（如 sayhello）仅使用 `ChainBuilder`，不需要 JWT 验证或 OpenFGA 鉴权。

## 目标 / 非目标

**目标：**
- 将 Authn 和 Authz 中间件移至 IAM 服务内部，消除 `convertAuthzRules` 桥接
- Authz 中间件直接使用 `iamv1.AuthzRuleEntry` 类型，无需独立定义镜像类型
- 在 `pkg/` 新增轻量 `IdentityFromHeader` 中间件，供网关架构下的其他服务使用
- `pkg/` 移除对 `openfga`、`redis`、proto 反射的直接依赖

**非目标：**
- 不改动 protoc-gen-servora-authz 插件（生成代码格式不变）
- 不改动 Authz 中间件的核心鉴权逻辑（仅移动位置和调整类型引用）
- 不改动 WhiteList / ChainBuilder / CORS（它们是通用工具，留在 `pkg/`）
- 不为 sayhello 添加 auth 中间件（仅为未来提供 `IdentityFromHeader` 能力）

## 决策

### 1. Authn/Authz 移到 IAM 的 internal/server/middleware/ 而非 internal/biz/

**选择**：`app/iam/service/internal/server/middleware/`
**替代方案**：`app/iam/service/internal/biz/middleware/` 或 `app/iam/service/internal/middleware/`

**理由**：
- Authn/Authz 是 Kratos transport 层中间件（`middleware.Middleware` 类型），语义上属于 server 层
- 与 `internal/server/http.go`、`grpc.go` 在同一层级，import 路径清晰
- DDD 分层中 biz 层不应包含 transport 相关的代码

### 2. Authz 中间件直接使用 `iamv1.AuthzRuleEntry` 而非定义新类型

**选择**：直接 import `iamv1.AuthzRuleEntry`
**替代方案 A**：在 `authzpb` 包中定义共享类型（需在 api/gen/ 中添加手写文件）
**替代方案 B**：在 IAM middleware 中定义新类型 + 接口

**理由**：
- 移到 IAM 内部后，Authz 中间件和生成代码属于同一 Go module（`app/iam/service`），可以直接 import
- 消除类型转换是本次重构的核心目标，引入新的中间类型会偏离目标
- `iamv1.AuthzRuleEntry` 结构已经足够清晰（Mode, Relation, ObjectType, IDField），无需包装
- 如果将来其他服务需要 Authz，可以从 IAM 提炼出共享包；现在遵循 YAGNI

### 3. `IdentityFromHeader` 放在 `pkg/` 而非各服务自行实现

**选择**：`pkg/transport/server/middleware/identity.go`
**替代方案**：各服务自己写 10 行代码读 header

**理由**：
- 所有非 IAM 服务获取用户身份的方式完全相同（读 `X-User-ID` 头 → 注入 actor）
- 放在 `pkg/` 可以统一 header key 名称，避免各服务拼写不一致
- 代码量虽小（~20 行），但包含 actor 注入逻辑，值得统一

### 4. `WhiteList` 保留在 `pkg/` 不动

**选择**：不移动
**替代方案**：随 Authn 一起移到 IAM

**理由**：
- `WhiteList` 是通用的 operation 匹配工具，不含 auth 逻辑
- 未来其他服务如需 selector 匹配（非 auth 场景），也可复用
- 当前 IAM 的 import 路径不变，减少改动量

### 5. `TokenFromContext` 和 `actor` 包保留在 `pkg/`

**选择**：`actor` 包和 `TokenFromContext` 留在 `pkg/`
**替代方案**：全部移到 IAM

**理由**：
- `actor` 包定义了 `Actor` 接口和 context 操作，是所有服务共享的身份抽象
- `IdentityFromHeader` 也需要 `actor.NewContext`，必须在 `pkg/` 层可用
- `TokenFromContext` 可能被其他服务用于 gRPC 传播 token

## 风险 / 权衡

| 风险 | 缓解措施 |
|------|----------|
| 将来新服务需要完整 Authn/Authz | 届时从 IAM 提炼出 `pkg/auth` 共享包，或直接在新服务中复制调整 |
| IAM 直接访问（绕过网关）时的安全性 | IAM 自己保留完整 JWT 验证，不受影响 |
| `IdentityFromHeader` 信任网关头的安全性 | 仅在容器网络内部使用；生产环境通过 Traefik 的 `trustForwardHeader: false` 防止客户端伪造 |
| `pkg/` 移除 authn/authz 后外部消费者 break | 当前无外部消费者；若有，Go module 的 major version bump 可以处理 |

## API 设计

### IdentityFromHeader 中间件

```go
// pkg/transport/server/middleware/identity.go

const DefaultUserIDHeader = "X-User-ID"

type IdentityOption func(*identityConfig)

type identityConfig struct {
    headerKey string
}

// WithHeaderKey 自定义 user ID 头名称，默认 "X-User-ID"。
func WithHeaderKey(key string) IdentityOption

// IdentityFromHeader 从网关注入的 HTTP 头读取用户身份。
// 适用于 ForwardAuth 架构下的非认证服务。
// 若 header 存在且非空，注入 UserActor 到 context；否则注入 AnonymousActor。
func IdentityFromHeader(opts ...IdentityOption) middleware.Middleware
```

### IAM 内部 Authz 中间件（移动后签名变化）

```go
// app/iam/service/internal/server/middleware/authz.go

// 不再需要独立的 AuthzRuleEntry 类型，直接使用 iamv1.AuthzRuleEntry
type AuthzOption func(*authzConfig)

func WithFGAClient(c *openfga.Client) AuthzOption
func WithAuthzRules(rules map[string]iamv1.AuthzRuleEntry) AuthzOption
func WithPlatformRootID(id string) AuthzOption
func WithAuthzCache(rdb *redis.Client, ttl time.Duration) AuthzOption
func Authz(opts ...AuthzOption) middleware.Middleware
```
