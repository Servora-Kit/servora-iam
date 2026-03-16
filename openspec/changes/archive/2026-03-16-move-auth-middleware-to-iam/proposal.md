## Why

Traefik ForwardAuth 接入后，微服务集群的认证统一由 IAM 服务负责。但 `pkg/transport/server/middleware/` 仍保留了完整的 Authn（JWT 验证）和 Authz（OpenFGA 权限检查）中间件，导致以下问题：

1. **`convertAuthzRules` 桥接代码**：生成代码 `iamv1.AuthzRuleEntry` 与 `pkg` 层 `svrmw.AuthzRuleEntry` 是结构相同但包不同的两个类型，每个服务都需要一个手写的转换函数
2. **`pkg/` 依赖过重**：共享库因 authz 中间件引入了 `openfga`、`redis`、proto 反射等重型依赖，但只有 IAM 在用
3. **其他服务不需要完整 auth 栈**：有了 ForwardAuth，sayhello 等服务只需读取网关注入的 `X-User-ID` 头即可获得用户身份，不需要 JWT 验证或 OpenFGA 检查
4. **proto 反射运行时风险**：`extractProtoField` 用字符串匹配字段名，字段重命名后只有运行时才能发现错误

现在做是因为 Traefik + ForwardAuth 刚接入完成，auth 架构已明确为"网关认证 + 服务授权"模式，是重新划分职责边界的最佳时机。

## What Changes

- **移动** `authn.go`、`authz.go` 从 `pkg/transport/server/middleware/` 到 `app/iam/service/internal/server/middleware/`
- **删除** `pkg/transport/server/middleware/` 中的 `authn.go` 和 `authz.go`
- **删除** IAM `server/http.go` 中的 `convertAuthzRules` 函数，中间件直接使用 `iamv1.AuthzRuleEntry`
- **新增** `pkg/transport/server/middleware/identity.go`：轻量 `IdentityFromHeader` 中间件，从网关 `X-User-ID` 头读取用户身份并注入 `actor.Actor`
- **修改** `WhiteList` 保留在 `pkg/` 中（通用工具，不含 auth 逻辑）

## Capabilities

### New Capabilities

- `identity-from-header`: 轻量中间件，从网关注入的 `X-User-ID` 头读取用户身份，供 ForwardAuth 架构下的非 IAM 服务使用

### Modified Capabilities

- `authz-middleware`: 从共享 `pkg` 层移至 IAM 服务内部，直接使用生成代码类型，消除 `convertAuthzRules`
- `authn-middleware`: 从共享 `pkg` 层移至 IAM 服务内部，IAM 作为认证中心独立管理 JWT 验证栈

## Impact

- **代码**：
  - 新增 `app/iam/service/internal/server/middleware/authn.go`
  - 新增 `app/iam/service/internal/server/middleware/authz.go`
  - 新增 `pkg/transport/server/middleware/identity.go`
  - 删除 `pkg/transport/server/middleware/authn.go`
  - 删除 `pkg/transport/server/middleware/authz.go`
  - 修改 `app/iam/service/internal/server/http.go`（去掉 `convertAuthzRules`，调整 import）
  - 修改 `app/iam/service/internal/server/grpc.go`（调整 import）
- **API**：无破坏性变更，对外接口不变
- **依赖**：`pkg/` 移除对 `openfga`、`redis`、proto 反射的直接依赖；`api/gen` module 无变化
- **BREAKING**：外部消费者如果引用了 `svrmw.Authn` 或 `svrmw.Authz` 需改为自行实现或拷贝，但当前无外部消费者
