## Why

servora 框架当前缺少统一的身份认证和授权服务。现有的认证逻辑分散在各个微服务内部，导致权限管理不一致、多租户支持薄弱、无法支持企业级的细粒度权限控制。随着框架的成熟和企业级应用场景的增加，需要一个独立的 IAM 服务来提供标准化的认证授权能力，支持三层多租户模型（Platform → Tenant → Workspace）和基于 ReBAC 的细粒度权限控制。

## Scope

- 在 `feature/iam-service` 分支上完成 IAM 服务重构与落地：认证、授权、多租户、OAuth、用户管理、OpenFGA 工具链。
- 统一对外提供 HTTP + gRPC 能力，以及 JWKS Endpoint 给其他服务验签。
- 明确通过 OpenSpec（proposal/design/tasks/spec）驱动实施，所有关键安全与可运维要求必须可验证。

## Out of Scope

- 本次不实现 LDAP/AD、SAML、完整 MFA（仅保留扩展位）。
- 本次不实现 Platform 管理 API（V1 固定 `platform:root`）。
- 本次不引入新的前端管理控制台。

## Implementation Strategy

**基于 example 分支重构**：为了减少工作量并复用现有基础设施，我们将在新分支 `feature/iam-service`（从 example 分支创建）中直接重构 `app/servora/service/` 为 IAM 服务，而不是新建独立服务。

**重构策略**：
- 删除 `app/sayhello/` 示例服务（简化项目结构）
- 删除 `app/servora/service/web/` 前端目录（IAM 服务不需要前端）
- 重命名 `app/servora/` 为 `app/iam/`
- 清理现有 Ent Schema，按 IAM 设计重新实现
- 更新所有配置文件（go.work, buf.yaml, Makefile, docker-compose, K8s manifests, AGENTS.md）

**优势**：
- 复用现有的 Docker、Makefile、配置文件结构
- 不破坏 example 分支的完整性（独立分支开发）
- 减少基础设施配置工作量
- 可以随时回退到 example 分支

## What Changes

- 重构 `app/servora/service/` 为 `app/iam/service/`，提供认证和授权能力
- 集成 OpenFGA 作为权限引擎，实现 Zanzibar 风格的 ReBAC 权限模型
- 实现三层多租户架构（Platform → Tenant → Workspace）
- 提供 JWKS Endpoint（`/.well-known/jwks.json`）供其他服务验证 JWT Token
- 抽象通用组件到 `pkg/`：
  - `pkg/authz/`：OpenFGA 客户端封装和权限中间件
  - `pkg/multitenancy/`：多租户上下文传播和数据隔离
  - `pkg/oauth/`：OAuth2/OIDC Provider 接口和实现
  - `pkg/ent/mixin/`：Ent Mixin（Time, SoftDelete, Tenant, Platform）
- 支持 HTTP + gRPC 双协议
- 支持邮箱注册/登录、OAuth2/OIDC 第三方登录（GitHub/Google/QQ）
- 支持混合注册模式（个人自助注册 + 企业管理员创建）
- 支持软删除和硬删除

## Capabilities

### New Capabilities

- `iam-authentication`: 用户认证能力，包括邮箱注册/登录、JWT Token 签发与验证、Refresh Token 机制、JWKS Endpoint
- `iam-authorization`: 基于 OpenFGA 的 ReBAC 权限检查，包括权限验证、关系元组管理、列表过滤
- `iam-multitenancy`: 三层多租户管理（Platform → Tenant → Workspace），V1 固定 Platform 为 `root`，用户只需管理 Tenant 和 Workspace；包括租户创建、成员管理、租户隔离
- `iam-oauth`: OAuth2/OIDC 第三方登录，包括 Provider 配置（Tenant 级别）、授权流程、账号绑定
- `iam-user-management`: 用户管理，包括用户 CRUD、跨租户关联、软删除/硬删除
- `iam-openfga-toolchain`: OpenFGA 工具链，包括模型验证、测试、部署、密钥轮换

### Modified Capabilities

<!-- 无现有能力需要修改 -->

## Acceptance Criteria

- `openspec validate iam-service --strict` 必须通过。
- 所有新增 Requirement 必须使用 MUST/SHALL 描述可验证约束。
- 授权降级策略必须默认 fail-closed，仅允许显式白名单端点豁免。
- OAuth 必须具备最小安全边界：PKCE、state/nonce、redirect_uri 精确匹配、`iss/aud/exp/nbf` 校验。
- 迁移任务必须具备回滚触发条件、回滚步骤与一致性对账检查。
- 审计与可观测性必须在 tasks 中有可执行条目并可验收。

## Impact

**重构服务**：
- `app/servora/service/` → `app/iam/service/`：完整的 IAM 微服务实现（V1 不包含 Platform 管理 API）
- 删除 `app/sayhello/service/`：移除示例服务
- 删除 `app/servora/service/web/`：移除前端目录

**新增共享库**：
- `pkg/authz/`：权限检查组件（供其他服务复用）
- `pkg/multitenancy/`：多租户基础设施（供其他服务复用）
- `pkg/oauth/`：OAuth2 客户端（供其他服务复用）
- `pkg/ent/mixin/`：Ent Mixin（供其他服务复用）

**基础设施依赖**：
- OpenFGA 服务（需要独立部署，依赖 PostgreSQL）
- Redis（Refresh Token 存储和权限缓存）

**工具链**：
- 新增 Makefile 命令：`make openfga.init`, `make openfga.model.validate`, `make openfga.model.test`
- 新增 OpenFGA 模型文件：`manifests/openfga/model/iam.fga`
- 新增 OpenFGA 测试文件：`manifests/openfga/tests/iam.fga.yaml`

**配置文件更新**：
- `go.work`：移除 sayhello，更新 servora 为 iam
- `buf.yaml`：移除 sayhello proto 路径，更新 servora 为 iam
- `Makefile`：移除 sayhello 相关命令，更新 servora 为 iam
- `docker-compose.yaml` / `docker-compose.dev.yaml`：保持 `docker-compose.yaml` 仅承载基础设施，更新 `docker-compose.dev.yaml` 中的开发服务定义（移除 sayhello，更新 servora 为 iam）
- `manifests/k8s/`：删除 sayhello 目录，重命名 servora 为 iam
- `AGENTS.md`：移除 sayhello 引用，更新 servora 为 iam，说明新的项目结构
- `README.md`：说明这是 IAM 开发分支

**其他服务影响**：
- 未来新增的服务可以通过 JWKS Endpoint 验证 IAM 签发的 JWT Token
- 通过 gRPC 调用 IAM 服务进行权限检查

**Go 依赖**：
- `github.com/openfga/go-sdk v0.7.5`
- `golang.org/x/oauth2 v0.36.0`
- `github.com/golang-jwt/jwt/v5 v5.3.1`
- `github.com/lestrrat-go/jwx/v3`（JWKS 实现）
- `entgo.io/ent v0.14.5`
- `github.com/redis/go-redis/v9 v9.18.0`

## Release Strategy

- 采用 Phase 0→4 渐进发布，每个阶段都要求“可回滚 + 可验收”后再进入下一阶段。
- 对高风险变更（授权模型、密钥轮换、迁移）采用先验证后切换策略。
- 发生高风险告警（授权异常率、OpenFGA 不可用、迁移对账失败）时立即停止推进并执行回滚预案。

## Backward Compatibility

- 对下游服务保持 JWKS 验签与 gRPC 调用的兼容接入路径。
- 在迁移窗口内通过双写/影子校验降低切换风险，避免一次性硬切换。
- 对已有分支结构保持隔离（main/example/feature），框架与服务改动边界清晰，避免破坏主干发布语义。
