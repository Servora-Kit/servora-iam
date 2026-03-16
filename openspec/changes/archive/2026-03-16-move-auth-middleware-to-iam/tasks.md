## 1. 创建 IAM 内部 middleware 包

- [x] 1.1 创建 `app/iam/service/internal/server/middleware/` 目录
- [x] 1.2 将 `pkg/transport/server/middleware/authn.go` 移动到 IAM 内部 middleware 包，调整 package 声明和 import 路径
- [x] 1.3 将 `pkg/transport/server/middleware/authz.go` 移动到 IAM 内部 middleware 包，将 `AuthzRuleEntry` 替换为 `iamv1.AuthzRuleEntry`，删除独立的 `AuthzRuleEntry` 类型定义
- [x] 1.4 验证 IAM 内部 middleware 包编译通过：`go build ./app/iam/service/internal/server/middleware/...`

## 2. 适配 IAM 服务 server 层

- [x] 2.1 修改 `app/iam/service/internal/server/http.go`：将 `svrmw.Authn`/`svrmw.Authz` 替换为 IAM 内部 middleware 的引用
- [x] 2.2 删除 `convertAuthzRules` 函数，Authz 中间件直接传入 `iamv1.AuthzRules`
- [x] 2.3 修改 `app/iam/service/internal/server/grpc.go`：调整 Authn/Authz 中间件引用，简化 `remapAuthzRulesForGRPC` 消除类型转换
- [x] 2.4 验证 IAM 服务编译通过：`go build ./app/iam/service/...`

## 3. 新增 IdentityFromHeader 中间件

- [x] 3.1 创建 `pkg/transport/server/middleware/identity.go`：实现 `IdentityFromHeader` 中间件，从 `X-User-ID` 头读取用户身份并注入 `actor.Actor` 到 context
- [x] 3.2 创建 `pkg/transport/server/middleware/identity_test.go`：测试有/无 header、空 header、自定义 key、无 transport 场景
- [x] 3.3 验证 pkg 编译和测试通过

## 4. 清理 pkg 层

- [x] 4.1 删除 `pkg/transport/server/middleware/authn.go`
- [x] 4.2 删除 `pkg/transport/server/middleware/authz.go`
- [x] 4.3 检查残留引用：发现 `pkg/transport/client/middleware/authn.go` 使用 `TokenFromContext`，提取为 `token.go` 保留在 pkg 层
- [x] 4.4 验证根 module 和 IAM 服务均编译通过
- [x] 4.5 运行 `go mod tidy`，确认 `pkg/` 不再直接依赖 `openfga`、`protoreflect`

## 5. 全局验证

- [x] 5.1 运行 `go build ./...` 确保所有 module 编译通过
- [x] 5.2 运行 `make test` 确保现有测试通过
- [x] 5.3 通过 curl 验证 ForwardAuth 流程正常（无 token → 401，有效 token → 204 + X-User-ID）
- [x] 5.4 运行 `make lint.go`，确认无新增 lint 问题（全部为已有问题）
