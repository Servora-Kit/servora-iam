# IAM OAuth Specification

## ADDED Requirements

### Requirement: The system MUST support OAuth provider configuration at tenant level

系统 MUST 支持在 Tenant 级别配置 OAuth Provider（GitHub/Google/QQ），每个 Tenant 可以配置自己的 Client ID 和 Client Secret。

#### Scenario: Create OAuth Provider configuration

- **WHEN** Tenant admin 创建 OAuth Provider 配置并提供 provider_type、client_id 和 client_secret
- **THEN** 系统创建 OAuthProvider 记录（tenant_id 关联到当前 Tenant）、返回配置信息（不包含 client_secret）

#### Scenario: Update OAuth Provider configuration

- **WHEN** Tenant admin 更新 OAuth Provider 的 client_id 或 client_secret
- **THEN** 系统更新 OAuthProvider 记录、返回更新后的信息

#### Scenario: Delete OAuth Provider configuration

- **WHEN** Tenant admin 删除 OAuth Provider 配置
- **THEN** 系统软删除 OAuthProvider 记录、保留已绑定的 OAuthAccount

#### Scenario: List OAuth Providers for tenant

- **WHEN** Tenant member 查询可用的 OAuth Provider
- **THEN** 系统返回当前 Tenant 配置的所有 Provider 列表

### Requirement: The system MUST implement OAuth authorization code flow

系统 MUST 支持标准的 OAuth 2.0 授权码流程（Authorization Code Flow）。

#### Scenario: Generate authorization URL

- **WHEN** 用户请求 OAuth 登录并指定 provider_type 和 tenant_slug
- **THEN** 系统生成授权 URL（包含 state 参数，编码 tenant_id 和 redirect_uri）、返回 URL

#### Scenario: Handle OAuth callback

- **WHEN** OAuth Provider 回调系统并提供 code 和 state
- **THEN** 系统验证 state、使用 code 交换 access_token、获取用户信息、创建或绑定 OAuthAccount、签发 JWT Token

#### Scenario: OAuth callback with invalid state

- **WHEN** OAuth Provider 回调时提供无效的 state
- **THEN** 系统返回错误 "Invalid state parameter"，HTTP 状态码 400

#### Scenario: OAuth callback with expired state

- **WHEN** OAuth Provider 回调时 state 已过期（超过 10 分钟）
- **THEN** 系统返回错误 "State expired"，HTTP 状态码 400

### Requirement: The system MUST enforce OAuth security boundaries

系统 MUST 对 OAuth 登录流程执行 PKCE、state/nonce、redirect_uri 精确匹配与 token 声明校验（`iss`、`aud`、`exp`、`nbf`）。

#### Scenario: OAuth callback without valid PKCE or nonce

- **WHEN** OAuth 回调缺少有效 PKCE 验证结果或 nonce 不匹配
- **THEN** 系统拒绝登录并返回错误 "Invalid OAuth security context"，HTTP 状态码 400

#### Scenario: OAuth callback with non-allowlisted redirect_uri

- **WHEN** 回调请求中的 redirect_uri 不在 Tenant 配置白名单内
- **THEN** 系统拒绝请求并返回错误 "Invalid redirect_uri"，HTTP 状态码 400

#### Scenario: OAuth provider token claim verification fails

- **WHEN** Provider 返回的 token 无法通过 `iss`、`aud`、`exp`、`nbf` 校验
- **THEN** 系统拒绝登录并返回错误 "Invalid provider token"，HTTP 状态码 401

### Requirement: The system MUST support OAuth account binding

系统 MUST 支持将 OAuth 账号绑定到现有用户。

#### Scenario: First-time OAuth login creates new user

- **WHEN** 用户首次使用 OAuth 登录且 OAuth 邮箱未注册
- **THEN** 系统创建新用户（email 来自 OAuth）、创建 OAuthAccount 记录、创建 Tenant 和默认 Workspace、签发 JWT Token

#### Scenario: OAuth login with existing email

- **WHEN** 用户使用 OAuth 登录且 OAuth 邮箱已注册
- **THEN** 系统创建 OAuthAccount 记录并绑定到现有用户、签发 JWT Token

#### Scenario: Bind OAuth account to logged-in user

- **WHEN** 已登录用户请求绑定 OAuth 账号
- **THEN** 系统执行 OAuth 流程、创建 OAuthAccount 记录并关联到当前用户

#### Scenario: Prevent duplicate OAuth account binding

- **WHEN** 用户尝试绑定已绑定到其他用户的 OAuth 账号
- **THEN** 系统返回错误 "OAuth account already bound to another user"，HTTP 状态码 409

### Requirement: The system MUST support multiple OAuth providers

系统 MUST 支持多个 OAuth Provider（GitHub、Google、QQ）。

#### Scenario: GitHub OAuth login

- **WHEN** 用户选择 GitHub 登录
- **THEN** 系统使用 GitHub OAuth API、获取用户信息（login, email, avatar_url）

#### Scenario: Google OAuth login

- **WHEN** 用户选择 Google 登录
- **THEN** 系统使用 Google OAuth API、获取用户信息（email, name, picture）

#### Scenario: QQ OAuth login

- **WHEN** 用户选择 QQ 登录
- **THEN** 系统使用 QQ OAuth API、获取用户信息（openid, nickname, figureurl）

### Requirement: The system MUST support OAuth account unlinking

系统 MUST 支持解绑 OAuth 账号。

#### Scenario: Unlink OAuth account

- **WHEN** 用户请求解绑 OAuth 账号
- **THEN** 系统删除 OAuthAccount 记录、返回成功

#### Scenario: Prevent unlinking last login method

- **WHEN** 用户尝试解绑最后一个登录方式（无密码且只有一个 OAuth 账号）
- **THEN** 系统返回错误 "Cannot unlink the last login method"，HTTP 状态码 400

### Requirement: The system MUST support OAuth token refresh

系统 MUST 支持刷新 OAuth Provider 的 access_token（如果 Provider 支持）。

#### Scenario: Refresh OAuth access token

- **WHEN** 系统需要使用 OAuth access_token 访问 Provider API 且 token 已过期
- **THEN** 系统使用 refresh_token 刷新 access_token、更新 OAuthAccount 记录

#### Scenario: OAuth refresh token expired

- **WHEN** OAuth refresh_token 已过期
- **THEN** 系统标记 OAuthAccount 为 "需要重新授权"、通知用户重新登录
