# IAM Service Implementation Tasks

## 0. 项目重构准备（基于 example 分支）

- [ ] 0.1 创建新分支 `feature/iam-service`（从 example 分支创建）
- [ ] 0.2 删除 `app/sayhello/` 服务目录
- [ ] 0.3 删除 `app/servora/service/web/` 前端目录
- [ ] 0.4 重命名 `app/servora/` 为 `app/iam/`
- [ ] 0.5 更新 `app/iam/service/go.mod` 模块路径（`github.com/Servora-Kit/servora/app/iam/service`）
- [ ] 0.6 更新根目录 `go.work`，移除 sayhello，更新 servora 为 iam
- [ ] 0.7 更新根目录 `buf.yaml`，移除 sayhello proto 路径，更新 servora 为 iam
- [ ] 0.8 更新根目录 `Makefile`，移除 sayhello 相关命令，更新 servora 为 iam
- [ ] 0.9 更新 `docker-compose.yaml`，保持仅包含基础设施服务
- [ ] 0.10 更新 `docker-compose.dev.yaml`，移除 sayhello 服务，更新 servora 为 iam
- [ ] 0.11 更新 `manifests/k8s/`，删除 sayhello 目录，重命名 servora 为 iam
- [ ] 0.12 更新 `README.md`，说明这是 IAM 开发分支，移除 sayhello 相关说明
- [ ] 0.13 更新 `AGENTS.md`：
  - 移除 sayhello 引用（顶层目录、关键文件、API/Proto、服务实现、部署、常用命令等所有章节）
  - 更新 servora 为 iam（所有路径、命令、说明）
  - 更新"项目概览"说明当前只有 iam 服务
  - 更新"当前目录约定"说明 iam 服务的结构
  - 更新"前端"章节，说明 iam 服务不包含前端
- [ ] 0.14 全局搜索并替换所有 `app/servora` 为 `app/iam`（import 路径、配置文件、脚本等）
- [ ] 0.15 全局搜索并删除所有 `app/sayhello` 引用（import 路径、配置文件、脚本、文档等）
- [ ] 0.16 清理 `app/iam/service/internal/data/ent/schema/` 中的现有 Schema（保留目录结构）
- [ ] 0.17 执行 `make gen` 验证配置正确性
- [ ] 0.18 执行 `make lint.go` 确保代码规范检查通过
- [ ] 0.19 提交重构变更：`git commit -m "refactor(app): rename servora to iam, remove sayhello and web"`

## 1. 项目初始化

- [ ] 1.1 创建 `app/iam/service/` 目录结构（cmd, internal, configs, api）
- [ ] 1.2 创建 `app/iam/service/go.mod`，配置模块依赖
- [ ] 1.3 创建 `app/iam/service/Makefile`，复用 `app.mk`
- [ ] 1.4 更新根目录 `go.work`，添加 `app/iam/service`
- [ ] 1.5 更新根目录 `buf.yaml`，添加 IAM proto 模块路径

## 2. Proto API 定义

- [ ] 2.1 创建 `app/iam/service/api/protos/iam/auth/service/v1/auth.proto`（认证 API）
- [ ] 2.2 JWKS Endpoint 已集成在 auth.proto 中
- [ ] 2.3 创建 `app/iam/service/api/protos/iam/user/service/v1/user.proto`（用户管理 API）
- [ ] 2.4 创建 `app/iam/service/api/protos/iam/tenant/service/v1/tenant.proto`（租户管理 API）
- [ ] 2.5 创建 `app/iam/service/api/protos/iam/workspace/service/v1/workspace.proto`（工作空间管理 API）
- [ ] 2.6 创建 `app/iam/service/api/protos/iam/authz/service/v1/authz.proto`（权限检查 API）
- [ ] 2.7 执行 `make gen` 生成 Go 代码到 `api/gen/go/`

## 3. Ent Mixin 实现

- [ ] 3.1 创建 `pkg/ent/mixin/time.go`（TimeMixin: created_at, updated_at）
- [ ] 3.2 创建 `pkg/ent/mixin/soft_delete.go`（SoftDeleteMixin: deleted_at, status）
- [ ] 3.3 创建 `pkg/ent/mixin/tenant.go`（TenantMixin: tenant_id + 索引）
- [ ] 3.4 创建 `pkg/ent/mixin/platform.go`（PlatformMixin: platform_id + 索引）
- [ ] 3.5 创建 `pkg/ent/mixin/operator.go`（OperatorMixin: created_by, updated_by，可选）

## 4. Ent Schema 定义

- [ ] 4.1 创建 `app/iam/service/internal/data/ent/schema/user.go`（User Schema）
- [ ] 4.2 创建 `app/iam/service/internal/data/ent/schema/tenant.go`（Tenant Schema，包含 platform_id 固定为 1）
- [ ] 4.3 创建 `app/iam/service/internal/data/ent/schema/workspace.go`（Workspace Schema）
- [ ] 4.4 创建 `app/iam/service/internal/data/ent/schema/tenant_member.go`（TenantMember Schema）
- [ ] 4.5 创建 `app/iam/service/internal/data/ent/schema/workspace_member.go`（WorkspaceMember Schema）
- [ ] 4.6 创建 `app/iam/service/internal/data/ent/schema/oauth_provider.go`（OAuthProvider Schema）
- [ ] 4.7 创建 `app/iam/service/internal/data/ent/schema/oauth_account.go`（OAuthAccount Schema）
- [ ] 4.8 执行 `make ent` 生成 Ent 代码

## 5. 数据层实现（Data Layer）

- [ ] 5.1 创建 `app/iam/service/internal/data/data.go`（Data 层初始化，数据库连接）
- [ ] 5.2 创建 `app/iam/service/internal/data/user.go`（UserRepo 实现）
- [ ] 5.3 创建 `app/iam/service/internal/data/tenant.go`（TenantRepo 实现）
- [ ] 5.4 创建 `app/iam/service/internal/data/workspace.go`（WorkspaceRepo 实现）
- [ ] 5.5 创建 `app/iam/service/internal/data/token_store.go`（TokenStore 实现，Redis）
- [ ] 5.6 初始化固定 Platform 记录（id=1, slug="root", type="system"）

## 6. 业务层实现（Biz Layer - Phase 1: 核心认证）

- [ ] 6.1 创建 `app/iam/service/internal/biz/auth.go`（AuthUsecase: 注册、登录、刷新）
- [ ] 6.2 创建 `pkg/jwt/rsa.go`（RSA JWT 生成与验证，支持 kid）
- [ ] 6.3 pkg/jwks 包已存在，验证功能完整性
- [ ] 6.4 创建 `app/iam/service/internal/biz/tenant.go`（TenantUsecase: 租户管理）
- [ ] 6.5 创建 `app/iam/service/internal/biz/workspace.go`（WorkspaceUsecase: 工作空间管理）
- [ ] 6.6 实现用户注册逻辑（自动创建 Tenant + 默认 Workspace，关联到 platform:root）

## 7. JWT 和 JWKS 实现

- [ ] 7.1 创建 `pkg/jwt/claims.go`（扩展 Claims，支持 tenant_id + workspace_id，包含 kid）
- [ ] 7.2 创建 `pkg/jwks/manager.go`（密钥管理器，支持密钥轮换）
- [ ] 7.3 创建 `pkg/jwks/endpoint.go`（JWKS Endpoint 实现，符合三条协议约束）
- [ ] 7.4 集成 `lestrrat-go/jwx` 生成 JWK
- [ ] 7.5 实现密钥轮换脚本 `scripts/rotate-jwks-key.sh`（三阶段轮换）

## 8. 服务层实现（Service Layer - Phase 1）

- [ ] 8.1 创建 `app/iam/service/internal/service/auth.go`（AuthService 实现）
- [ ] 8.2 创建 `app/iam/service/internal/service/tenant.go`（TenantService 实现）
- [ ] 8.3 创建 `app/iam/service/internal/service/workspace.go`（WorkspaceService 实现）

## 9. HTTP/gRPC Server 配置

- [ ] 9.1 创建 `app/iam/service/internal/server/http.go`（HTTP Server，注册路由）
- [ ] 9.2 创建 `app/iam/service/internal/server/grpc.go`（gRPC Server）
- [ ] 9.3 创建 `app/iam/service/internal/server/middleware/auth.go`（JWT 认证中间件）
- [ ] 9.4 注册 JWKS Endpoint 路由（`GET /.well-known/jwks.json`）

## 10. 依赖注入（Wire）

- [ ] 10.1 创建 `app/iam/service/internal/server/wire.go`（Wire Provider Set）
- [ ] 10.2 创建 `app/iam/service/internal/biz/wire.go`（Wire Provider Set）
- [ ] 10.3 创建 `app/iam/service/internal/data/wire.go`（Wire Provider Set）
- [ ] 10.4 创建 `app/iam/service/cmd/iam/wire.go`（Wire 入口）
- [ ] 10.5 执行 `make wire` 生成 `wire_gen.go`

## 11. 配置文件

- [ ] 11.1 创建 `app/iam/service/configs/local/bootstrap.yaml`（本地开发配置，含数据库、Redis、JWT 基础设施配置）
- [ ] 11.2 创建 `app/iam/service/configs/local/biz.yaml`（IAM 业务专属配置，如 OpenFGA 地址、限流策略等）
- [ ] 11.3 创建 `app/iam/service/configs/docker/bootstrap.yaml`（容器环境地址覆盖）
- [ ] 11.4 创建 `app/iam/service/configs/docker/biz.yaml`（容器环境业务配置）
- [ ] 11.5 为业务配置定义专属 proto：`app/iam/service/api/protos/iam/conf/v1/biz_conf.proto`
- [ ] 11.6 执行 `make gen` 生成配置代码

## 12. 主程序入口

- [ ] 12.1 创建 `app/iam/service/cmd/iam/main.go`（主程序入口）
- [ ] 12.2 实现服务启动逻辑（HTTP + gRPC 双协议）
- [ ] 12.3 实现优雅关闭

## 13. OpenFGA 集成（Phase 2）

- [ ] 13.1 创建 `manifests/openfga/model/iam.fga`（OpenFGA 权限模型）
- [ ] 13.2 创建 `manifests/openfga/tests/iam.fga.yaml`（OpenFGA 模型测试）
- [ ] 13.3 创建 `pkg/authz/client.go`（OpenFGA 客户端封装）
- [ ] 13.4 创建 `pkg/authz/checker.go`（权限检查接口）
- [ ] 13.5 创建 `pkg/authz/middleware.go`（权限中间件）
- [ ] 13.6 创建 `scripts/openfga-init.sh`（OpenFGA 初始化脚本）
- [ ] 13.7 创建 `scripts/openfga-apply-model.sh`（OpenFGA 模型应用脚本）

## 14. 多租户基础设施（Phase 2）

- [ ] 14.1 创建 `pkg/multitenancy/context.go`（租户上下文传播）
- [ ] 14.2 创建 `pkg/multitenancy/middleware.go`（租户隔离中间件）
- [ ] 14.3 创建 `pkg/multitenancy/filter.go`（Ent 数据过滤器）
- [ ] 14.4 集成到 HTTP/gRPC Server

## 15. 权限检查实现（Phase 2）

- [ ] 15.1 实现 AuthzService（权限检查 API）
- [ ] 15.2 实现关系元组管理（WriteRelation, DeleteRelation）
- [ ] 15.3 实现列表过滤（ListObjects + Redis 会话游标分页：第一次调用获取所有 ID 存入 Redis，后续调用从 Redis 读取指定范围）
- [ ] 15.4 实现权限缓存（Redis）
- [ ] 15.5 集成权限中间件到所有需要鉴权的 API

## 16. OAuth2/OIDC 实现（Phase 3）

- [ ] 16.1 创建 `pkg/oauth/provider.go`（Provider 接口）
- [ ] 16.2 创建 `pkg/oauth/providers/github.go`（GitHub Provider）
- [ ] 16.3 创建 `pkg/oauth/providers/google.go`（Google Provider）
- [ ] 16.4 创建 `pkg/oauth/providers/qq.go`（QQ Provider）
- [ ] 16.5 实现 OAuth 授权流程（GetAuthURL, Callback）
- [ ] 16.6 实现 OAuth 账号绑定
- [ ] 16.7 实现 Tenant 级 Provider 配置管理

## 17. 软删除和硬删除（Phase 4）

- [ ] 17.1 实现 DeleteUser API（软删除）
- [ ] 17.2 实现 PurgeUser API（硬删除 + OpenFGA 清理）
- [ ] 17.3 实现 RestoreUser API（恢复）
- [ ] 17.4 实现 DeleteTenant API（软删除 + 级联）
- [ ] 17.5 实现 PurgeTenant API（硬删除 + 级联 + OpenFGA 清理）
- [ ] 17.6 实现 DeleteWorkspace API（软删除）
- [ ] 17.7 实现 PurgeWorkspace API（硬删除 + OpenFGA 清理）

## 18. Docker 和部署

- [ ] 18.1 更新根 `docker-bake.hcl`，为 IAM 服务新增构建 target
- [ ] 18.2 更新 `docker-compose.dev.yaml`，为 IAM 开发容器补充服务定义
- [ ] 18.3 更新 `docker-compose.yaml`，添加 OpenFGA 等基础设施服务
- [ ] 18.4 创建 `manifests/k8s/iam/` 部署清单
- [ ] 18.5 创建 `manifests/k8s/openfga/` 部署清单

## 19. Makefile 和工具链

- [ ] 19.1 在根 Makefile 添加 `openfga.init` 命令
- [ ] 19.2 在根 Makefile 添加 `openfga.model.validate` 命令
- [ ] 19.3 在根 Makefile 添加 `openfga.model.test` 命令
- [ ] 19.4 在根 Makefile 添加 `openfga.model.apply` 命令
- [ ] 19.5 在根 Makefile 添加 `jwks.rotate` 命令

## 20. 测试

- [ ] 20.1 编写 User 注册/登录单元测试
- [ ] 20.2 编写 JWT Token 签发/验证单元测试
- [ ] 20.3 编写 JWKS Endpoint 单元测试
- [ ] 20.4 编写 Tenant/Workspace 管理单元测试（Platform 固定为 root，无需测试 Platform CRUD）
- [ ] 20.5 编写 OpenFGA 权限检查单元测试
- [ ] 20.6 编写集成测试（使用 testcontainers）
- [ ] 20.7 执行 `make test` 确保所有测试通过

## 21. 文档

- [ ] 21.1 编写 `app/iam/service/README.md`（服务说明）
- [ ] 21.2 编写 `docs/iam/quickstart.md`（快速开始）
- [ ] 21.3 编写 `docs/iam/api.md`（API 文档）
- [ ] 21.4 编写 `docs/iam/openfga.md`（OpenFGA 使用指南）
- [ ] 21.5 编写 `docs/iam/jwks.md`（JWKS 使用指南）
- [ ] 21.6 编写 `docs/iam/migration.md`（迁移指南）

## 22. 验证和冒烟测试

- [ ] 22.1 执行 `make gen` 确保所有代码生成成功
- [ ] 22.2 执行 `make wire` 确保依赖注入成功
- [ ] 22.3 执行 `make lint.go` 确保代码规范检查通过
- [ ] 22.4 执行 `make test` 确保所有测试通过
- [ ] 22.5 启动 IAM 服务，验证 HTTP 和 gRPC 端口正常监听
- [ ] 22.6 调用注册 API，验证用户注册成功
- [ ] 22.7 调用登录 API，验证 Token 签发成功
- [ ] 22.8 访问 JWKS Endpoint，验证公钥返回正常
- [ ] 22.9 调用权限检查 API，验证 OpenFGA 集成正常
- [ ] 22.10 执行 `make compose.dev`，验证完整环境启动成功
- [ ] 22.11 验证 OpenFGA 不可用时受保护接口默认 fail-closed（返回 503），仅白名单端点可访问
- [ ] 22.12 验证 OAuth 安全约束（PKCE、state/nonce、redirect_uri 精确匹配、iss/aud/exp/nbf 校验）
- [ ] 22.13 验证迁移回滚预案可执行（触发条件、回滚步骤、对账检查）
- [ ] 22.14 验证审计事件完整性（登录、授权拒绝、关系变更、密钥轮换）

## 23. 安全性与可运维闭环

- [ ] 23.1 为 Open Questions 建立闭环任务清单（owner、due date、decision output）
- [ ] 23.2 实现 IAM API 限流中间件（Redis + Kratos），并配置分级限流策略
- [ ] 23.3 实现审计日志落地（认证、授权、关系变更、管理员操作、密钥轮换）
- [ ] 23.4 实现 OpenFGA/JWKS 关键指标上报（延迟、错误率、缓存命中率、验签失败率）
- [ ] 23.5 配置关键告警阈值（P99、错误率、连续失败次数）并完成告警联调
- [ ] 23.6 实现迁移影子校验/双写一致性检查工具并产出校验报告
- [ ] 23.7 增加 OpenFGA 模型版本切换与回滚演练任务（authorization_model_id 可回退）
