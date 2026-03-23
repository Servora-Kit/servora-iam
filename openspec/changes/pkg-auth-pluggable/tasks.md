## 1. Proto 治理：AuthzMode 迁移

- [x] 1.1 创建 `api/protos/servora/authz/v1/authz.proto`，将 `AuthzMode` 枚举和 `AuthzRule` message 从 `app/iam/service/api/protos/servora/authz/service/v1/authz.proto` 移入
- [x] 1.2 更新 `buf.yaml` 确保新 proto 路径被 Buf workspace 正确发现
- [x] 1.3 更新 `app/iam/service/api/protos/servora/authz/service/v1/authz.proto`，移除已迁移的定义，保留服务级别的内容（如有）
- [x] 1.4 运行 `make api` 重新生成 Go 代码，验证 `api/gen/go/servora/authz/v1/` 产出正确
- [x] 1.5 更新 `protoc-gen-servora-authz` 的 import path，生成代码引用 `servora/authz/v1` 而非 `servora/authz/service/v1`

## 2. pkg/authn 接口化

- [x] 2.1 在 `pkg/authn/authn.go` 中定义 `Authenticator` 接口和 `Server()` 中间件函数签名
- [x] 2.2 创建 `pkg/authn/jwt/` 子目录，迁移 JWT 验证逻辑：`jwt.go`（NewAuthenticator）、`claims.go`（ClaimsMapper + DefaultClaimsMapper + KeycloakClaimsMapper）、`options.go`
- [x] 2.3 从 `DefaultClaimsMapper` 中移除 Keycloak 特有映射（`iss→Realm`），将其放入 `KeycloakClaimsMapper`
- [x] 2.4 创建 `pkg/authn/noop/noop.go`（NoopAuthenticator）
- [x] 2.5 更新 `pkg/authn/authn.go`：移除旧的 `Authn()` 函数和相关 options，保留 `Server()` + 中间件级 options（`WithErrorHandler`）
- [x] 2.6 更新 `pkg/authn/authn_test.go` 适配新 API

## 3. pkg/authz 接口化

- [x] 3.1 在 `pkg/authz/authz.go` 中定义 `Authorizer` 接口、`DecisionDetail` 结构体、`Server()` 中间件函数签名
- [x] 3.2 定义 `WithDecisionLogger(fn)` option，替代 `WithAuditRecorder`
- [x] 3.3 更新 `AuthzRule.Mode` 字段引用新的共享 proto `authz/v1.AuthzMode`
- [x] 3.4 创建 `pkg/authz/openfga/` 子目录：`openfga.go`（NewAuthorizer 封装 `pkgopenfga.Client`）、`options.go`（WithRedisCache）
- [x] 3.5 创建 `pkg/authz/noop/noop.go`（NoopAuthorizer）
- [x] 3.6 从 `pkg/authz/authz.go` 中移除对 `pkg/openfga`、`pkg/audit`、`pkg/redis` 的直接依赖
- [x] 3.7 更新 `pkg/authz/authz_test.go` 适配新 API

## 4. protoc-gen-servora-authz 适配

- [x] 4.1 更新 `cmd/protoc-gen-servora-authz/main.go` 中生成代码的 import path（`servora/authz/v1` 替代 `servora/authz/service/v1`）
- [x] 4.2 运行 `make api` 验证所有服务的 `authz_rules.gen.go` 正确引用新 import path

## 5. 服务适配

- [x] 5.1 更新 `app/iam/service/internal/server/grpc.go` 和 `http.go`：使用 `authn.Server(jwt.NewAuthenticator(...))` 和 `authz.Server(openfga.NewAuthorizer(...))`
- [x] 5.2 更新 `app/iam/service/internal/server/` 中的 Wire provider（如需）
- [x] 5.3 更新 `app/sayhello/service/internal/server/grpc.go`：适配新 authn/authz API
- [x] 5.4 更新 `app/audit/service/internal/server/grpc.go`：适配新 authz API（审计服务使用 DecisionLogger 回调桥接 audit.Recorder）

## 6. 构建验证与清理

- [x] 6.1 运行 `make api` 确保所有 proto 生成代码正确
- [x] 6.2 运行 `go build ./...` 验证所有模块编译通过
- [x] 6.3 运行 `make test` 验证测试通过
- [x] 6.4 运行 `make lint.go` 验证无 lint 错误
- [x] 6.5 删除 `app/iam/service/api/protos/servora/authz/service/v1/authz.proto` 中已迁移的定义（如步骤 1.3 遗留空文件则删除整个文件）

## 7. 文档与提交

- [x] 7.1 更新主设计文档 `docs/plans/2026-03-20-keycloak-openfga-audit-design.md`，记录此变更的完成状态和新架构约束
- [x] 7.2 更新 `pkg/authn/AGENTS.md` 和 `pkg/authz/AGENTS.md` 反映新的目录结构和接口
- [x] 7.3 提交代码（分步 commit：proto 迁移 → authn 接口化 → authz 接口化 → 服务适配 → 清理）
