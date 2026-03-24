# 设计文档：Servora Keycloak 接入

**日期：** 2026-03-24
**前身：** 2026-03-20 认证/授权/审计框架演进（Phase 1–3.5 已归档）
**状态：** 规划中

---

## 前置工作归档

Phase 1–3.5（框架骨架、审计全链路、代码生成、authn/authz 接口化）已全部完成，详见：
- **归档文档**：`docs/plans/archive/2026-03-20-framework-audit-authz-phases.md`
- **OpenSpec 归档**：`openspec/changes/archive/` 下的 5 个已归档 change
- **沉淀 specs（21 个）**：`openspec/specs/` 目录

**已完成的前置清理**：
- 删除 `Actor.Scope(key)` / `SetScope()` / `ScopeFromContext()` 死代码
- 删除 `OrgScopeMixin`（硬编码 `organization_id` 的业务特化 mixin）
- 修复 `KeycloakClaimsMapper`：补全 `realm_access.roles` 合并逻辑
- 清理 `pkg/` 中所有业务特化代码和文档引用

---

## 背景

Servora 框架层的认证、授权、审计基础设施已就绪：
- `pkg/authn`：可插拔 `Authenticator` 接口，现有 `jwt/`（含 `KeycloakClaimsMapper`）和 `noop/` 引擎
- `pkg/authz`：可插拔 `Authorizer` 接口，现有 `openfga/` 和 `noop/` 引擎
- `pkg/audit`：全链路审计（Kafka → Audit Service → ClickHouse）
- `pkg/actor`：通用 principal 模型（user/service/anonymous/system）

下一步是接入 **Keycloak** 作为认证中心，完成从自建 IAM 到外部 IdP 的切换。

---

## 核心决策

| 决策点 | 结论 |
|---|---|
| 认证中心 | **Keycloak**（OIDC/OAuth2 IdP） |
| Keycloak 职责 | **仅负责 AuthN**：用户认证、token 签发、JWKS、粗粒度 realm roles |
| 资源级授权 | **OpenFGA**（运行时实时查询 tuples，不从 JWT 读取） |
| Realm 策略 | **单一 `servora` realm**，不同平台/应用用不同 client |
| 多租户 | **暂不做**，`Actor.Realm()` 预留口子 |
| 角色策略 | **先只用 Realm Roles**，Client Roles 后续按需引入 |
| 网关认证 | 网关统一验 token → 注入 principal header → 业务服务信任 header |
| 网关选型 | 先用 **Traefik**，但保持可插拔（配置化，不硬绑） |
| 业务服务验 JWT | 默认**不重复验**，信任网关 header；`pkg/authn/jwt/` 保留用于直连/服务间调用 |
| IAM 服务 | **保留**作为示例服务，不清理 |
| 前端 | **暂不对接**，当前无需运行前端 |
| actor 模型 | `Keycloak claims → 网关 header → HeaderAuthenticator → actor.Actor` |

---

## 权限三层模型

```
┌──────────────────────────────────────────────────────────────────┐
│  Layer 1: Keycloak JWT Claims                                   │
│  ─────────────────────────────                                  │
│  身份 + 粗粒度角色（登录时签发，token 有效期内不变）               │
│                                                                  │
│  sub, email, realm_access.roles: ["platform-admin", "developer"]│
│  scope: "openid profile email"                                  │
└──────────────────────────┬───────────────────────────────────────┘
                           │ 网关提取 → headers
                           ▼
┌──────────────────────────────────────────────────────────────────┐
│  Layer 2: actor.Actor                                           │
│  ────────────────────                                           │
│  标准化的 principal 投影（框架内部统一模型）                       │
│                                                                  │
│  ID(), Type(), Email(), Subject(), Realm(), Roles(), Scopes()   │
│  Attrs() — 扩展属性 bag                                         │
└──────────────────────────┬───────────────────────────────────────┘
                           │ 业务代码调用
                           ▼
┌──────────────────────────────────────────────────────────────────┐
│  Layer 3: OpenFGA Tuples                                        │
│  ───────────────────────                                        │
│  细粒度资源级权限（运行时实时查询，权限变更即时生效）               │
│                                                                  │
│  user:abc | editor | project:proj-42                            │
│  authz.Check(actor, "file.delete", "project:proj-42")           │
└──────────────────────────────────────────────────────────────────┘
```

**关键原则**：
- Keycloak JWT 里**不放** OpenFGA tuples — 避免 token 膨胀和权限过时
- Keycloak realm roles 是**身份标签**（"你是谁"），OpenFGA 是**资源权限**（"你能做什么"）
- 两者互补，不冲突

---

## 认证引擎对比

| 引擎 | 包路径 | 工作方式 | 适用场景 |
|------|--------|---------|---------|
| JWT | `pkg/authn/jwt/` | 服务自己验签 JWT、解析 claims → actor | 无网关 / 服务间调用 |
| Header | `pkg/authn/header/`（Phase 2 新建） | 从网关注入的 headers → actor | 有网关，服务信任 headers |
| Noop | `pkg/authn/noop/` | 返回 anonymous actor | 开发/测试跳过认证 |

两种认证引擎的输出完全一致（`actor.Actor`），下游 `pkg/authz` 和 `pkg/audit` 透明。

---

## Actor 类型与 Keycloak 映射

| Actor Type | 来源 | Keycloak 关系 |
|------------|------|---------------|
| `TypeUser` | 用户登录获取的 access token | Keycloak 签发（Authorization Code / Password Grant） |
| `TypeService` | 服务间调用的 client credentials token | Keycloak 签发（Client Credentials Grant） |
| `TypeSystem` | 代码内部构造（如定时任务、系统操作） | 与 Keycloak 无关 |
| `TypeAnonymous` | 无 token 或认证跳过 | 与 Keycloak 无关 |

---

## 职责分工

```
┌──────────┐    OIDC     ┌──────────┐   principal   ┌────────────────┐
│ Keycloak │◄───────────►│  网关     │──headers──────►│  业务服务       │
│          │  验 token    │(Traefik) │               │ pkg/authn/header│
└──────────┘             └──────────┘               │ → actor.Actor  │
                                                     │ pkg/authz      │
                                                     │ → OpenFGA      │
                                                     │ pkg/audit      │
                                                     │ → Kafka        │
                                                     └────────────────┘
```

- **Keycloak**：用户认证、OIDC/OAuth2、token 签发、JWKS、realm roles 管理
- **网关**：统一入口、对接 Keycloak 验 token、将 principal 注入上游 header
- **业务服务**：从 header 构建 actor → `pkg/authz` 授权（OpenFGA 实时查询） → 审计 emit

---

## 分阶段计划

### Phase 1：Keycloak 基础设施

**目标**：开发环境一键启动 Keycloak，OIDC endpoints 可用。

**核心任务**：
1. docker-compose 新增 Keycloak 服务（`quay.io/keycloak/keycloak`，`start-dev` 模式）
2. 创建 realm 初始化文件 `manifests/keycloak/servora-realm.json`：
   - `servora` realm
   - OAuth2 clients：`servora-gateway`（网关用）、`servora-web`（前端用，预留）
   - Realm Roles：`platform-admin`、`developer`、`viewer`
   - 测试用户：admin（platform-admin 角色）、dev（developer 角色）
3. 使用 `--import-realm` 挂载到 `/opt/keycloak/data/import/` 实现自动初始化
4. 验证 OIDC discovery endpoint (`/.well-known/openid-configuration`) 和 JWKS endpoint 可用

**不做**：
- 不对接网关
- 不改动 pkg 代码
- 不修改现有服务
- 不做多租户
- 不配置 Client Roles

### Phase 2：pkg/authn/header/ 引擎

**目标**：实现 `HeaderAuthenticator`，从网关注入的 header 构造 `actor.Actor`。

**核心任务**：
1. 创建 `pkg/authn/header/` 子目录
2. 实现 `HeaderAuthenticator` 实现 `authn.Authenticator` 接口
3. 从 header 映射 actor 字段：
   - `X-User-ID` → `Actor.ID()`
   - `X-Subject` → `Actor.Subject()`
   - `X-Client-ID` → `Actor.ClientID()`
   - `X-Principal-Type` → `Actor.Type()`
   - `X-Realm` → `Actor.Realm()`
   - `X-Email` → `Actor.Email()`
   - `X-Roles` → `Actor.Roles()`
   - `X-Scopes` → `Actor.Scopes()`
4. 支持 `WithHeaderMapping` 自定义 header 名称
5. `X-Principal-Type` 决定 actor 类型（user/service/anonymous）

**已有 spec**：`openspec/specs/identity-header-enhancement/spec.md`（已更新为 HeaderAuthenticator 方向）

**不做**：
- 不迁移或删除现有代码
- 不修改网关配置

### Phase 3：网关认证集成

**目标**：网关对接 Keycloak，完成 token 验证 → principal header 注入 → 业务服务 authn 的完整链路。

**核心任务**：
1. Traefik 配置对接 Keycloak OIDC（ForwardAuth 或 OIDC plugin）
2. 验证链路：用户登录 → Keycloak 签发 token → 请求带 token → 网关验证 → 注入 principal header → 业务服务 `authn.Server(headerAuth)` → actor in context
3. 保持网关可插拔：认证配置不硬编码在业务服务或 pkg 中

**依赖**：Phase 1（Keycloak 可用）+ Phase 2（HeaderAuthenticator 可用）

**不做**：
- 不清理 IAM 中的 issuer 能力
- 不对接前端

---

## 未来方向

- **Servora 生态扩展**：`pkg/broker` 补更多实现（NATS/RabbitMQ）、`pkg/task`/`pkg/queue` 任务队列、统一 observability
- **前端对接**：Keycloak 登录流程（当需要前端时再规划）
- **IAM 演进**：保留作为示例服务，可能逐步演化为管理控制台
- **多租户**：在单一 realm 基础上，通过 Keycloak Organization 特性或自定义 claims 实现

---

## 约束继承

本阶段继承 Phase 1–3.5 确立的所有实现约束（详见归档文档 `docs/plans/archive/2026-03-20-framework-audit-authz-phases.md` 中的"实现约束"章节）。Keycloak 接入相关的新约束将在各 Phase 实现时补充。
