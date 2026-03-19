# IAM 服务重构：从后台管理系统到统一身份平台

Date: 2026-03-18

## 目标

将 IAM 服务从"多租户后台管理系统"转型为"统一身份平台"，作为 servora 微服务生态中所有业务平台（视频、购物、云管理等）的身份与认证基础设施。

## 核心决策

| 决策点 | 结论 |
|---|---|
| 用户群 | 统一账号，一个用户走遍所有平台 |
| Tenant / Organization | 从 IAM 中移除，各业务平台自己管理组织概念 |
| 授权 | IAM 提供 OpenFGA 基础设施，平台直连 OpenFGA 做 check，IAM 不代理授权请求 |
| M2M 服务间身份 | 认证（Client Credentials grant）+ 授权（OpenFGA service 间关系）|
| 前端管理界面 | 保留为轻量管理控制台（用户管理 + 应用管理）|
| 登录页 | 独立 `web/accounts/`，与 `web/iam/` 通过 `web/ui/` 共享 shadcn/ui + Catppuccin 组件 |
| RBAC | 全部移除，User.role 字段（admin/user）即可 |
| Authz proto 注解 | 枚举改为字符串，泛化后所有下游服务复用同一套 proto 注解 + 代码生成 + 中间件链路 |

## IAM 的三个职责

1. **身份管理** — 用户是谁（User CRUD、注册、资料）
2. **认证服务** — 证明你是谁（登录、SSO、token 签发、M2M）
3. **授权基础设施** — OpenFGA 运维 + pkg 封装（不理解业务语义）

## 1. 核心实体

### User

账户字段（独立列，可查询/索引）：

| 字段 | 类型 | 说明 |
|---|---|---|
| id | UUID v7 | 全局唯一身份标识，对应 OIDC `sub` |
| username | string, unique | 登录标识，对应 OIDC `preferred_username` |
| email | string, unique | 登录凭证 |
| email_verified | bool | |
| email_verified_at | time, nullable | |
| phone | string, optional | |
| phone_verified | bool | |
| password | string | bcrypt hash |
| role | string | "admin" \| "user" |
| status | string | "active" \| "disabled" |
| created_at | time | |
| updated_at | time | |

个人资料（JSON 列，嵌入 `oidc.UserInfoProfile`）：

| 字段 | 说明 |
|---|---|
| name | 显示名 |
| given_name | 名 |
| family_name | 姓 |
| nickname | 昵称 |
| picture | 头像 URL |
| gender | |
| birthdate | |
| zoneinfo | |
| locale | 语言偏好 |

Go 层直接嵌入 `*oidc.UserInfoProfile`，签发 ID Token 时零映射。

### Application（OAuth2 客户端）

| 字段 | 类型 | 说明 |
|---|---|---|
| id | UUID v7 | |
| client_id | string | |
| client_secret_hash | string | bcrypt hash |
| name | string | |
| type | string | "web" \| "native" \| "m2m" |
| redirect_uris | []string | web/native 需要，m2m 为空 |
| scopes | []string | |
| grant_types | []string | web: authorization_code; m2m: client_credentials |
| access_token_type | string | |
| id_token_lifetime | duration | |
| created_at | time | |
| updated_at | time | |

### 移除的实体

Tenant, Organization, OrganizationMember, Dict, DictType, DictItem, Position, RbacRole, RbacPermission, RbacPermissionGroup, RbacPermissionAPI, RbacPermissionMenu, RbacMenu, RbacRolePermission, RbacUserRole。

## 2. 认证架构

### 用户认证（人 → 平台）

标准 OIDC 授权码流程：

```
用户访问视频平台 → 302 到 IAM 授权端点 → web/accounts/ 登录页
→ IAM 签发 authorization_code → 302 回到视频平台 callback
→ 视频平台用 code 换 access_token + id_token
```

SSO 体验：已登录用户访问其他平台时，OIDC `prompt=none` 静默签发 code，用户无感知（< 500ms 跳转）。

zitadel/oidc Provider 已支持此流程，需确保：
- OIDC Storage 实现中去掉对 Tenant 的依赖
- `prompt=none` 静默登录正常工作
- ID Token claims 携带 `profile` JSON 中的字段

### M2M 认证（服务 → 服务）

复用 OIDC Client Credentials Grant：

```
微服务 → POST /oauth/token (grant_type=client_credentials, client_id, client_secret)
→ IAM 验证 → 签发 M2M JWT (sub=app:{client_id}, type=m2m)
→ 下游服务验 JWT 签名（JWKS）+ OpenFGA 关系检查
```

需在 OIDC Storage 中实现 `ClientCredentialsTokenRequest` 接口。

### Token 策略

| Token | 用途 | 生命周期 |
|---|---|---|
| ID Token | 用户身份信息 | 可配置（默认 1h） |
| Access Token | API 调用凭证 | 15-30min |
| Refresh Token | 换新 access token | 7-30d |
| M2M Token | 服务间调用凭证 | 1h |

### JWKS

所有下游服务通过 `GET /.well-known/jwks.json` 获取公钥离线验签，无需每次请求问 IAM。

## 3. 授权基础设施

### 架构

平台直连 OpenFGA，IAM 不代理授权请求：

```
各业务平台 ──→ OpenFGA（直接 check/write）
IAM 负责：管理 OpenFGA store/model、提供 pkg/ 封装
```

### OpenFGA Model

```fga
model
  schema 1.1

type user

type platform
  relations
    define admin: [user]

type service
  relations
    define caller: [service]
```

各业务平台自行扩展（如 `type video`、`type server`），通过 model 文件合并。

### Authz Proto 注解（泛化）

`AuthzRule` message 中 `mode` 保持枚举（简化为 NONE/CHECK），`relation` 和 `object_type` 改为字符串以支持各平台自定义：

```protobuf
enum AuthzMode {
  AUTHZ_MODE_UNSPECIFIED = 0;
  AUTHZ_MODE_NONE = 1;   // 跳过鉴权
  AUTHZ_MODE_CHECK = 2;  // 做 OpenFGA check
}

message AuthzRule {
  AuthzMode mode = 1;     // 枚举：NONE | CHECK
  string relation = 2;    // 各平台自定义，如 "can_delete", "admin"
  string object_type = 3; // 各平台自定义，如 "video", "server"
  string id_field = 4;    // 请求中的资源 ID 字段名
}
```

### 全服务统一链路

```
authz.proto（通用注解）
  → protoc-gen-servora-authz（代码生成）
  → authz_rules.gen.go（每个服务各自生成）
  → pkg/authz.Middleware（通用中间件，查 OpenFGA）
```

所有下游服务在 proto RPC 上标注 authz rule，`make api` 后自动生成规则代码。

### M2M 授权

OpenFGA 中管理 service 间调用关系：

```
service:payment-service#caller@service:order-service
```

## 4. 前端架构

### 两个应用

| 应用 | 定位 | 域名（示例） |
|---|---|---|
| `web/accounts/` | 面向所有用户的公开页面（登录/注册/回调/验证/重置） | accounts.servora.dev |
| `web/iam/` | 仅管理员使用的管理控制台（用户管理/应用管理） | admin.servora.dev |

### UI 组件共享

`web/ui/` 作为 pnpm workspace 包（`@servora/ui`），包含 shadcn/ui 组件 + Catppuccin Latte/Mocha 主题 token。两个应用通过 `"@servora/ui": "workspace:*"` 引用，保证视觉统一。

### web/accounts/ 页面

login, register, callback, verify-email, reset-password。

### web/iam/ 精简后

保留：dashboard（概览）, users/（用户管理）, applications/（应用管理）, settings/（个人资料、改密码）。

删除：tenants, organizations, rbac, positions, dict 及相关组件/store/hook。

### OIDC 登录页迁移

`internal/oidc/login.go` 中的 Go template 登录页移除，OIDC Provider 的 login_url 指向 `web/accounts/` 的 URL。

## 5. pkg/ 变更

| 包 | 操作 | 说明 |
|---|---|---|
| `pkg/actor` | 修改 | 移除 TenantID(), OrganizationID() |
| `pkg/jwks` | 保留 | |
| `pkg/jwt` | 保留 | |
| `pkg/openfga` | 保留 | |
| `pkg/authn` | 新增 | JWT 验签中间件，整合 jwks + jwt |
| `pkg/authz` | 新增 | 通用 OpenFGA check 中间件，从 IAM internal 上提 |

## 6. 清理范围

### Proto

删除 tenant/, organization/, dict/, position/, rbac/ 目录及对应的 i_*.proto。

### Ent Schema

删除 14 个 schema 文件（tenant, organization, organization_member, dict_type, dict_item, position, 8 个 rbac_* schema）。修改 user.go、application.go。重新 `make gen.ent`。

### Biz / Data / Service

删除对应模块的 usecase, repo, service 实现。ProviderSet 精简为 Authn + User + Application。

### Server / 中间件

Authz 中间件移除 Tenant/Org resolve 逻辑，简化为通用 check。路由注册移除已删服务。

### OpenFGA

Model 重写为 user + platform + service。测试用例对应更新。

### protoc-gen-servora-authz

适配字符串字段替代枚举。
