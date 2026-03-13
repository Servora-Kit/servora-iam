# IAM P0 手动测试步骤

本文档描述邮箱验证、密码重置（Token 链接）、GetUser 的手动测试流程。前置：本地已能跑 IAM 服务并连接 Mailpit / Redis / Postgres。

---

## 一、环境准备

### 1. 启动基础设施（含 Mailpit）

在**项目根目录**执行：

```bash
make compose.up
```

确认以下服务可用：

- Postgres: `localhost:5432`
- Redis: `localhost:6379`
- Mailpit SMTP: `localhost:1025`（收信）
- Mailpit Web UI: http://localhost:8025（查看邮件）

### 2. 启动 IAM 服务

```bash
cd app/iam/service
# 若需指定配置目录
go run ./cmd/server -conf ./configs/local
```

或使用根目录：

```bash
make compose.dev   # 若 Makefile 中有 dev 目标并会启动 IAM
```

确认 IAM HTTP 端口（默认 `8000`），下文以 `http://localhost:8000` 为例。

### 3. 可选：数据库与 OpenFGA

若刚改过 User schema（如新增 `email_verified`），按当前策略需**重置数据库**后再测：

```bash
make compose.reset
make compose.up
# 再启动 IAM，Ent 会自动建表
```

---

## 二、测试 1：邮箱验证（RequestEmailVerification + VerifyEmail）

### 步骤 1.1 注册一个测试用户

```bash
curl -s -X POST http://localhost:8000/v1/auth/signup/using-email \
  -H "Content-Type: application/json" \
  -d '{"name":"testuser","password":"123456","password_confirm":"123456","email":"test@example.com"}'
```

预期：返回 JSON 含 `id`、`name`、`email`、`role`。

### 步骤 1.2 请求发送验证邮件

```bash
curl -s -X POST http://localhost:8000/v1/auth/request-email-verification \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com"}'
```

预期：`{"success":true}`（无论该邮箱是否已验证都返回成功，防枚举）。

### 步骤 1.3 在 Mailpit 中查看邮件

1. 打开 http://localhost:8025
2. 应看到一封主题为 “Verify your email” 的邮件
3. 打开邮件，正文中有一个链接，形如：`http://localhost:3000/verify-email?token=<一串 hex>`
4. 从 URL 中复制 `token` 参数值（整段，不含 `token=` 前后空格）

### 步骤 1.4 调用验证接口（提交 token）

将 `<TOKEN>` 替换为上一步复制的 token：

```bash
curl -s -X POST http://localhost:8000/v1/auth/verify-email \
  -H "Content-Type: application/json" \
  -d '{"token":"<TOKEN>"}'
```

预期：`{"success":true}`。

### 步骤 1.5 再次用同一 token 调用验证接口

同一 token 再请求一次：

```bash
curl -s -X POST http://localhost:8000/v1/auth/verify-email \
  -H "Content-Type: application/json" \
  -d '{"token":"<同一个TOKEN>"}'
```

预期：HTTP 4xx，错误信息为 token 无效或已过期（一次性消费）。

### 步骤 1.6 确认用户已标记为邮箱已验证

使用下面「测试 3」的登录 + GetUser 步骤，查看该用户的 `email_verified` 应为 `true`。

---

## 三、测试 2：密码重置（RequestPasswordReset + ResetPassword）

### 步骤 2.1 请求重置密码（不泄露是否存在该邮箱）

```bash
curl -s -X POST http://localhost:8000/v1/auth/request-password-reset \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com"}'
```

预期：始终返回 `{"success":true}`（即使邮箱不存在也返回成功，防枚举）。

### 步骤 2.2 在 Mailpit 中获取重置链接里的 token

1. 打开 http://localhost:8025，查看最新一封 “Reset your password” 邮件
2. 从邮件中的链接里复制 `token` 参数，例如：  
   `http://localhost:3000/reset-password?token=<TOKEN>`

### 步骤 2.3 提交新密码完成重置

将 `<TOKEN>` 替换为上面复制的 token，新密码需满足 5～10 位（与当前 Signup 校验一致）：

```bash
curl -s -X POST http://localhost:8000/v1/auth/reset-password \
  -H "Content-Type: application/json" \
  -d '{"token":"<TOKEN>","new_password":"654321","new_password_confirm":"654321"}'
```

预期：`{"success":true}`。

### 步骤 2.4 用新密码登录

```bash
curl -s -X POST http://localhost:8000/v1/auth/login/email-password \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"654321"}'
```

预期：返回 `access_token`、`refresh_token`、`expires_in`。

### 步骤 2.5 旧密码不可再登录

```bash
curl -s -X POST http://localhost:8000/v1/auth/login/email-password \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"123456"}'
```

预期：401 或错误信息表示密码错误。

### 步骤 2.6 同一 token 不能重复使用

再次用**同一个**重置 token 调用 ResetPassword（可随便填一个新密码）：

```bash
curl -s -X POST http://localhost:8000/v1/auth/reset-password \
  -H "Content-Type: application/json" \
  -d '{"token":"<同一个TOKEN>","new_password":"999999","new_password_confirm":"999999"}'
```

预期：4xx，提示 token 无效或已过期。

---

## 四、测试 3：GetUser（含 email_verified）

### 步骤 3.1 登录获取 access_token

```bash
curl -s -X POST http://localhost:8000/v1/auth/login/email-password \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"654321"}'
```

从响应中复制 `access_token`，记为 `<ACCESS_TOKEN>`。

### 步骤 3.2 查当前用户信息（/v1/user/info）

```bash
curl -s http://localhost:8000/v1/user/info \
  -H "Authorization: Bearer <ACCESS_TOKEN>"
```

预期：返回当前用户 `id`、`name`、`email`、`role`（及 `email_verified` 若接口已返回）。

### 步骤 3.3 GetUser 查自己（用上面返回的 id）

将 `<USER_ID>` 替换为步骤 3.2 返回的 `id`：

```bash
curl -s http://localhost:8000/v1/users/<USER_ID> \
  -H "Authorization: Bearer <ACCESS_TOKEN>"
```

预期：返回对应用户信息，且包含 `email_verified` 字段（若已做邮箱验证则为 `true`）。

### 步骤 3.4 GetUser 查他人（无权限）

若存在另一用户 B 的 `USER_B_ID`，用普通用户 A 的 token 请求 B 的信息：

```bash
curl -s http://localhost:8000/v1/users/<USER_B_ID> \
  -H "Authorization: Bearer <ACCESS_TOKEN_A>"
```

预期：403，提示权限不足（除非 A 为 admin/operator）。

### 步骤 3.5 GetUser 用 admin/operator 查任意用户

使用具有 admin 或 operator 角色的用户登录，取其 `access_token`，再请求任意用户 id：

```bash
curl -s http://localhost:8000/v1/users/<任意用户ID> \
  -H "Authorization: Bearer <ADMIN_ACCESS_TOKEN>"
```

预期：200，返回该用户信息（含 `email_verified`）。

---

## 五、检查清单小结

| 场景 | 预期 |
|------|------|
| 请求邮箱验证 | 始终 200，未验证则发邮件 |
| 验证邮件中链接 token | 一次有效，再次请求 4xx |
| 请求密码重置 | 始终 200，存在用户则发邮件 |
| 重置链接 token + 新密码 | 一次有效，登录用新密码成功 |
| 旧密码 / 旧 refresh_token | 登录失败或 refresh 失效 |
| GetUser 自己 | 200，含 email_verified |
| GetUser 他人（普通用户） | 403 |
| GetUser 他人（admin/operator） | 200 |

---

## 六、常见问题

- **收不到邮件**：确认 IAM 配置里 `mail.smtp` 指向 `127.0.0.1:1025`（本地）或 `mailpit:1025`（docker 内），且 `make compose.up` 已启动 Mailpit。
- **verify/reset 返回 4xx**：检查 token 是否从邮件链接中完整复制（无多余空格、换行），且未使用过。
- **GetUser 404**：确认 id 为有效 UUID，且对应用户存在（未软删）。
- **密码长度**：当前 Signup/Reset 校验为 5～10 位，与 `authn.proto` 中 `min_len/max_len` 一致。
