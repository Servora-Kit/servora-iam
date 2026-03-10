# IAM Service Design

## Context

servora 框架当前的认证授权能力分散在各个微服务内部，缺少统一的身份管理和权限控制。现有实现（app/servora/service/internal/biz/auth.go）仅支持简单的 JWT + RBAC，无法满足企业级多租户和细粒度权限控制的需求。

**当前状态**：
- 认证逻辑：邮箱+密码登录，JWT Token 签发（Access + Refresh）
- 授权逻辑：基于角色的简单权限检查（guest/user/admin/operator）
- 多租户：无支持
- 权限粒度：角色级别，无资源级别权限控制

**实施策略**：
- 基于 example 分支创建新分支 `feature/iam-service`
- 重构 `app/servora/service/` 为 `app/iam/service/`（而不是新建独立服务）
- 删除 `app/sayhello/` 和 `app/servora/service/web/`（简化项目结构）
- 复用现有的 Docker、Makefile、配置文件结构
- 清理现有 Ent Schema，按 IAM 设计重新实现

**约束**：
- 必须遵循 servora 的 DDD 分层架构（service → biz → data）
- 必须支持 HTTP + gRPC 双协议
- 必须使用 Proto First 开发方式
- 必须将通用组件抽象到 pkg/ 供其他服务复用
- 必须使用 Ent ORM（与 servora 保持一致）
- 必须更新所有配置文件以反映项目结构变化

**利益相关者**：
- 框架使用者：需要开箱即用的 IAM 能力
- 未来的微服务：需要通过 JWKS 验证 Token，通过 gRPC 调用权限检查
- 企业用户：需要多租户隔离和细粒度权限控制

## Goals / Non-Goals

**Goals:**
- 提供统一的认证授权服务，支持邮箱登录和 OAuth2/OIDC 第三方登录
- 实现三层多租户模型（Platform → Tenant → Workspace），支持企业级组织架构
- 集成 OpenFGA 实现 ReBAC 权限模型，支持细粒度的资源级权限控制
- 提供 JWKS Endpoint，供其他服务验证 JWT Token
- 抽象通用组件到 pkg/，提高代码复用性
- 支持软删除和硬删除，满足合规要求
- 提供完善的 OpenFGA 工具链（模型验证、测试、部署）

**Non-Goals:**
- 不实现 LDAP/AD 集成（可作为未来扩展）
- 不实现 SAML 协议（专注于 OAuth2/OIDC）
- 不实现 MFA（多因素认证）在 Phase 1（预留接口）
- 不实现 Session 管理（纯 JWT，无状态）
- 不实现密码策略配置（使用固定策略）

## Decisions

### 1. 三层多租户模型：Platform → Tenant → Workspace（V1 运营两层化）

**决策**：采用三层数据模型，但 **V1 固定单一 Platform**（`platform:root`），不开放 Platform 管理 API。

**理由**：
- **保留扩展性**：数据模型支持三层，未来可平滑升级到完整的企业集团场景
- **降低复杂度**：V1 用户无需理解 Platform 概念，只需关注 Tenant 和 Workspace
- **运营两层化**：用户注册时自动关联到 `platform:root`，实际使用体验是两层（Tenant → Workspace）
- **避免过早优化**：等出现真实的集团级场景再开放 Platform CRUD

**V1 实现策略**：
- 数据库中存在 Platform 表，但只有一条记录：`id=1, slug="root", type="system"`
- 所有 Tenant 的 `platform_id` 固定为 `1`
- OpenFGA 关系中所有用户自动关联到 `platform:root`
- 不提供 Platform 创建/更新/删除 API
- 用户注册流程：创建 User → 创建 Tenant（platform_id=1）→ 创建默认 Workspace

**替代方案**：
- 两层模型（Tenant → Workspace）：数据模型无法扩展，未来升级需要迁移
- 完整三层模型：V1 复杂度过高，用户学习成本大

**影响**：
- 数据模型保留 Platform 实体（但 V1 固定为 root）
- OpenFGA 关系链保持三层（Platform → Tenant → Workspace → Resource）
- 用户注册流程简化（无需创建 Platform）
- V1 不提供 Platform 管理 API（Phase 5 或更晚再开放）

### 2. OpenFGA 作为权限引擎

**决策**：使用 OpenFGA 实现 ReBAC 权限模型，而不是自研或使用 Casbin。

**理由**：
- OpenFGA 是 Google Zanzibar 的开源实现，经过大规模验证
- 支持关系型权限模型，天然适合多租户场景
- 支持权限继承（Platform → Tenant → Workspace）
- 提供 gRPC API，性能优秀
- 社区活跃，文档完善

**替代方案**：
- Casbin：更轻量，但不支持关系型权限模型，无法实现复杂的权限继承
- 自研：开发成本高，稳定性和性能无法保证
- Ory Keto：功能类似 OpenFGA，但社区较小

**影响**：
- 需要独立部署 OpenFGA 服务（依赖 PostgreSQL）
- 需要学习 OpenFGA 的 DSL 语法（.fga 文件）
- 需要实现密钥轮换和缓存机制

### 3. JWKS Endpoint 而不是 Service Account

**决策**：提供 JWKS Endpoint（`/.well-known/jwks.json`）供其他服务验证 JWT，而不是实现 Service Account / M2M 认证。

**理由**：
- JWKS 是 OAuth2/OIDC 标准，兼容性好
- 其他服务可以无状态验证 JWT，无需调用 IAM 服务
- 支持密钥轮换而不中断服务
- OpenStack Keystone 等成熟系统采用此方案

**替代方案**：
- Service Account + Client Credentials Flow：需要实现额外的 OAuth2 流程，增加复杂度
- 共享密钥：不支持密钥轮换，安全性差

**影响**：
- 需要实现密钥管理和轮换机制
- 需要使用 `lestrrat-go/jwx` 库生成 JWK
- 其他服务需要定期拉取 JWKS 并缓存

### 4. Ent Mixin 复用字段

**决策**：使用 Ent Mixin 机制复用通用字段（CreatedAt, UpdatedAt, DeletedAt, TenantID, PlatformID）。

**理由**：
- 避免在每个 Schema 中重复定义相同字段
- 统一时间戳和软删除的实现方式
- 提高代码可维护性
- go-wind-admin 已验证此方案的可行性

**替代方案**：
- 手动在每个 Schema 中定义：重复代码多，容易遗漏
- 使用代码生成：增加工具链复杂度

**影响**：
- 需要创建 `pkg/ent/mixin/` 包
- 所有 Schema 需要实现 `Mixin()` 方法

### 5. OAuth Provider 配置在 Tenant 级别

**决策**：OAuth Provider 配置（Client ID/Secret）存储在 Tenant 级别，而不是 Platform 或全局级别。

**理由**：
- 不同部门可能有不同的 OAuth 应用（如不同的 GitHub Organization）
- 提供最大的灵活性
- 符合企业级 IAM 的需求（如 Casdoor）

**替代方案**：
- Platform 级别：所有 Tenant 共享配置，灵活性不足
- 全局级别：无法支持企业自定义 OAuth 应用
- 混合模式：实现复杂度高

**影响**：
- 需要在 OAuthProvider 表中增加 TenantID 字段
- OAuth 回调时需要解析 state 获取 TenantID

### 6. 邮箱登录优先，OAuth 靠后

**决策**：Phase 1 实现邮箱注册/登录，Phase 3 实现 OAuth2/OIDC 第三方登录。

**理由**：
- 邮箱登录是基础功能，优先级最高
- OAuth 实现复杂度较高，需要更多测试
- 可以先验证核心架构（多租户 + OpenFGA）

**影响**：
- Phase 1 无法使用 GitHub/Google 登录
- 需要预留 OAuth 的数据模型和接口

### 7. 软删除 + 硬删除

**决策**：支持软删除（标记 deleted_at）和硬删除（物理删除），而不是只支持软删除。

**理由**：
- 软删除：满足合规要求（审计、恢复）
- 硬删除：满足 GDPR 等数据删除要求
- 提供灵活性，由调用方决定删除方式

**替代方案**：
- 只支持软删除：无法满足 GDPR 要求
- 只支持硬删除：无法恢复误删除的数据

**影响**：
- 需要在 Mixin 中实现 SoftDeleteMixin
- 需要在 API 中提供 DeleteUser 和 PurgeUser 两个接口
- 硬删除需要清理 OpenFGA 关系元组

## Risks / Trade-offs

### 1. OpenFGA 性能风险

**风险**：三层权限继承可能导致权限检查性能下降，尤其是列表过滤场景。

**缓解措施**：
- **权限缓存**：使用 Redis 缓存权限检查结果，按接口敏感度区分 TTL
  - 敏感接口（删除、权限变更）：TTL 1 分钟
  - 普通接口（查询、列表）：TTL 5 分钟
- **列表分页**：使用**应用侧游标（Redis 会话游标）**实现分页
  - OpenFGA 的 `StreamedListObjects` 无服务端 continuation token
  - 第一次调用：获取所有 ID 并存入 Redis（key: `cursor:{sessionID}`, TTL 10 分钟）
  - 后续调用：从 Redis 读取并返回指定范围的 ID
  - 参考 Kemate 的 `internal/openfga/authorizer.go` 实现
- **性能监控**：监控 OpenFGA 的响应时间，设置告警阈值（P99 < 100ms）

### 2. 密钥轮换复杂度

**风险**：JWKS 密钥轮换需要三阶段部署（分发 → 切换 → 清理），操作复杂，容易出错。

**缓解措施**：
- 提供自动化脚本（`scripts/rotate-jwks-key.sh`）
- 在 Makefile 中提供命令（`make jwks.rotate`）
- 文档中详细说明轮换流程和回滚方案
- 设置密钥过期告警（提前 7 天）

### 3. 三层模型学习成本

**风险**：三层模型（Platform → Tenant → Workspace）比两层模型复杂，用户理解成本高。

**缓解措施**：
- **V1 运营两层化**：固定 Platform 为 `root`，用户只需理解 Tenant 和 Workspace
- 提供详细的文档和示例
- 提供 CLI 命令快速创建测试数据（`svr iam seed`）
- 在 API 响应中包含层级关系（如 Workspace 响应中包含 Tenant 信息）
- 文档中明确说明：V1 不需要关心 Platform，所有 Tenant 自动属于 `platform:root`

### 4. OpenFGA 依赖风险

**风险**：OpenFGA 服务故障会导致所有权限检查失败，影响业务可用性。

**缓解措施**：
- OpenFGA 部署高可用（至少 2 个实例）
- 使用 Redis 缓存权限检查结果（降级策略）
- 监控 OpenFGA 健康状态，设置告警
- 默认采用 fail-closed：OpenFGA 不可用时拒绝受保护请求（返回 503）
- 仅允许显式白名单端点（如健康检查）在故障时绕过鉴权

### 5. 数据迁移风险

**风险**：现有服务迁移到 IAM 服务时，需要迁移用户数据和权限关系，可能导致数据不一致。

**缓解措施**：
- 提供数据迁移脚本（`scripts/migrate-to-iam.sh`）
- 支持双写模式（同时写入旧系统和 IAM 服务）
- 提供数据一致性校验工具
- 分阶段迁移（先迁移认证，再迁移授权）

## Security and Reliability Guardrails

- **授权不可隐式降级**：任何依赖故障场景默认 fail-closed，不允许“静默放行全部请求”。
- **OAuth 安全边界固定**：授权码流程必须校验 PKCE、state、nonce，且 `redirect_uri` 精确匹配白名单；Token 验证必须检查 `iss`、`aud`、`exp`、`nbf`。
- **密钥轮换可观测**：密钥分发/切换/清理三阶段必须记录审计事件并配置告警（过期、切换失败、下游验签失败率上升）。
- **高风险路径强一致**：撤权、成员变更、删除等敏感路径优先使用高一致性权限查询策略，并对缓存失效做显式处理。

## Observability and Audit Baseline

- 认证指标：登录成功率/失败率、Token 刷新成功率、JWKS 命中率与下游验签失败率。
- 授权指标：Check/ListObjects 延迟（P50/P95/P99）、拒绝率、OpenFGA 错误率、缓存命中率。
- 审计事件最小集合：登录成功/失败、权限决策拒绝、关系元组变更、租户级管理员操作、密钥轮换动作。
- 告警阈值：默认以 P99、错误率、连续失败次数为触发条件，触发后阻断下一阶段发布。

## Migration Gates and Rollback Criteria

- 每个 Phase 进入下一阶段前必须满足：功能验收通过 + 关键指标稳定 + 审计事件完整。
- 回滚触发条件至少包括：授权错误率异常、迁移对账失败、核心依赖不可用且超出恢复窗口。
- 回滚执行必须包含：配置回退、模型版本回退（authorization_model_id）、数据对账复核。
- 迁移期间采用影子校验/双写策略，确保切换前后权限判定一致性可证明。

## Migration Plan

### Phase 0: 项目重构准备（1 天）

**目标**：完成项目结构重构，为 IAM 实现做准备。

**步骤**：
1. 创建新分支 `feature/iam-service`（从 example 分支创建）
2. 删除 `app/sayhello/` 和 `app/servora/service/web/`
3. 重命名 `app/servora/` 为 `app/iam/`
4. 更新所有配置文件（go.work, buf.yaml, Makefile, docker-compose, K8s manifests, AGENTS.md, README.md）
5. 全局替换所有 `app/servora` 和 `app/sayhello` 引用
6. 执行 `make gen` 和 `make lint.go` 验证配置正确性
7. 提交重构变更

**验证**：
- 项目可以正常编译
- 所有配置文件引用正确
- Git 历史清晰

### Phase 1: 核心认证 + 基础设施（2-3 周）

**目标**：实现基础的认证能力和数据模型。

**步骤**：
1. 清理现有 `app/iam/service/internal/data/ent/schema/`，按 IAM 设计重新创建
2. 定义 Proto API（auth, user, tenant, workspace, authz）
3. 实现 Ent Schema 和 Mixin（包含 Platform，但 V1 固定为 root）
4. 实现邮箱注册/登录（自动创建 Tenant，关联到 `platform:root`）
5. 实现 JWT Token 签发和验证（Claims 包含 `kid`）
6. 实现 JWKS Endpoint（符合三条协议约束）
7. 部署 OpenFGA（docker-compose）

**验证**：
- 用户可以注册和登录
- 可以获取 Access Token 和 Refresh Token
- 未来的服务可以通过 JWKS 验证 Token

### Phase 2: 多租户 + 授权（2-3 周）

**目标**：实现三层多租户（V1 运营两层化）和 OpenFGA 权限检查。

**步骤**：
1. 初始化固定 Platform（`platform:root`，id=1）
2. 实现 Tenant/Workspace 管理 API（不包含 Platform API）
3. 实现用户-租户关联
4. 定义 OpenFGA 权限模型（.fga 文件，三层继承）
5. 实现 OpenFGA 客户端封装（`pkg/authz/`）
6. 实现权限检查中间件
7. 实现关系元组管理 API
8. 实现列表过滤（基于 OpenFGA ListObjects + Redis 会话游标）

**验证**：
- 用户可以创建 Tenant/Workspace（自动关联到 `platform:root`）
- 用户可以邀请其他用户加入 Tenant
- 权限检查正常工作（owner/admin/member/viewer）
- 列表过滤正常工作（只返回有权限的资源，支持分页）

### Phase 3: OAuth2/OIDC（1-2 周）

**目标**：实现第三方登录。

**步骤**：
1. 实现 OAuth Provider 配置 API（Tenant 级别）
2. 实现 OAuth 授权流程（GitHub/Google/QQ）
3. 实现 OAuth 账号绑定
4. 实现 `pkg/oauth/` 通用组件

**验证**：
- 用户可以通过 GitHub/Google/QQ 登录
- OAuth 账号可以绑定到现有用户
- Tenant 管理员可以配置自己的 OAuth 应用

### Phase 4: 资源管理（1 周）

**目标**：实现软删除和硬删除。

**步骤**：
1. 实现软删除 API（DeleteUser, DeleteTenant, DeleteWorkspace）
2. 实现硬删除 API（PurgeUser, PurgeTenant, PurgeWorkspace）
3. 实现恢复 API（RestoreUser, RestoreTenant, RestoreWorkspace）
4. 实现级联删除逻辑
5. 实现 OpenFGA 关系元组清理

**验证**：
- 软删除后数据仍存在，状态为 deleted
- 硬删除后数据物理删除，OpenFGA 关系元组清理
- 恢复后数据状态恢复为 active

### 回滚策略

**Phase 0 回滚**：
- 切换回 example 分支：`git checkout example`
- 删除 feature/iam-service 分支：`git branch -D feature/iam-service`

**Phase 1 回滚**：
- 回退到 Phase 0 完成时的提交
- 停止 OpenFGA 服务

**Phase 2 回滚**：
- 禁用权限检查中间件（环境变量 `IAM_AUTHZ_ENABLED=false`）
- 回退到 Phase 1 状态

**Phase 3 回滚**：
- 禁用 OAuth 登录入口
- 回退到 Phase 2 状态

**Phase 4 回滚**：
- 禁用删除 API
- 回退到 Phase 3 状态

## Open Questions

1. **密钥轮换周期**：JWKS 密钥应该多久轮换一次？建议 30-90 天，需要根据安全要求确定。

2. **权限缓存 TTL**：Redis 缓存权限检查结果的 TTL 应该设置多久？建议按接口敏感度区分：敏感接口 1 分钟，普通接口 5 分钟。

3. **OpenFGA 部署方式**：生产环境是否需要独立部署 OpenFGA 集群？建议至少 2 个实例，需要评估成本。

4. **MFA 支持**：是否需要在 Phase 1 预留 MFA 接口？建议预留，但不实现。

5. **LDAP/AD 集成**：是否需要在未来支持 LDAP/AD 集成？建议作为 Phase 5，需要评估需求优先级。

6. **审计日志**：是否需要记录所有认证和授权操作的审计日志？建议在 Phase 2 实现，存储到 ClickHouse。

7. **API 限流**：是否需要对 IAM API 进行限流？建议在 Phase 2 实现，使用 Redis + Kratos 限流中间件。

## Open Questions Closure Policy

- 每个 Open Question 必须映射到一个任务项，包含 owner、截止时间、备选方案和决策输出（accept/defer/drop）。
- 未决问题不得跨阶段漂移：进入下一 Phase 前必须关闭当前 Phase 的高风险问题。
- 对“延期实现”的问题必须记录触发条件和最晚重新评估时间。
