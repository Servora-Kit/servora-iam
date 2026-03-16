# PurgeUser 级联删除 集成测试步骤

本文档描述 PurgeUser 级联删除的端到端验证流程。前置：本地已运行 IAM 服务（含 Postgres、Redis、OpenFGA）。

> 对应变更：`openspec/changes/purge-user-cascade/`

---

## 一、环境准备

### 1. 启动全部基础设施 + IAM

```bash
make compose.dev
```

### 2. 初始化 OpenFGA（首次或重置后）

```bash
make openfga.init
# 然后重建 IAM 容器使其读取新的 FGA_STORE_ID/FGA_MODEL_ID
docker compose -f docker-compose.yaml -f docker-compose.dev.yaml up -d iam
```

### 3. 确认 IAM 可访问

```bash
curl -s http://localhost:8000/healthz
# 预期: {"status":"ok"} 或 200
```

### 4. Docker keys 目录

若 `app/iam/service/configs/docker/keys/` 为空，需复制密钥：

```bash
cp app/iam/service/configs/keys/iam.rsa.pem app/iam/service/configs/docker/keys/
docker restart iam
```

---

## 二、构造测试数据

### 步骤 1：注册测试用户

```bash
curl -s -X POST http://localhost:8000/v1/auth/signup/using-email \
  -H "Content-Type: application/json" \
  -d '{"name":"purgetest","email":"purgetest@test.com","password":"Test1234!","password_confirm":"Test1234!"}'
```

预期：返回 `id`、`name`、`email`。记录 `USER_ID`。

### 步骤 2：登录获取 Token

```bash
curl -s -X POST http://localhost:8000/v1/auth/login/email-password \
  -H "Content-Type: application/json" \
  -d '{"email":"purgetest@test.com","password":"Test1234!"}'
```

预期：返回 `accessToken`。记录为 `TOKEN`。

### 步骤 3：创建 Organization（用户自动成为 owner）

```bash
curl -s -X POST http://localhost:8000/v1/organizations \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"purge-test-org","slug":"purge-test-org"}'
```

预期：返回 `organization.id`。记录为 `ORG_ID`。

### 步骤 4：在该 Organization 下创建 Project

```bash
curl -s -X POST http://localhost:8000/v1/projects \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Organization-ID: $ORG_ID" \
  -d '{"name":"purge-test-proj","slug":"purge-test-proj"}'
```

预期：返回 `project.id`。记录为 `PROJ_ID`。

### 步骤 5：确认数据库中数据完整

```bash
docker exec servora_db psql -U servora -d iam -c "
  SELECT 'user' AS type, id::text, name FROM users WHERE id='$USER_ID'
  UNION ALL
  SELECT 'org', id::text, name FROM organizations WHERE id='$ORG_ID'
  UNION ALL
  SELECT 'proj', id::text, name FROM projects WHERE id='$PROJ_ID';
"
```

预期：3 行数据（user、org、proj）。

```bash
docker exec servora_db psql -U servora -d iam -c "
  SELECT 'org_member' AS type, role, organization_id::text FROM organization_members WHERE user_id='$USER_ID'
  UNION ALL
  SELECT 'proj_member', role, project_id::text FROM project_members WHERE user_id='$USER_ID';
"
```

预期：至少 2 行 org_member（含默认 org + purge-test-org）和至少 1 行 proj_member。

---

## 三、执行 PurgeUser

### 步骤 6：用 admin 登录

```bash
curl -s -X POST http://localhost:8000/v1/auth/login/email-password \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@servora.dev","password":"changeme"}'
```

预期：返回 `accessToken`。记录为 `ADMIN_TOKEN`。

### 步骤 7：调用 PurgeUser

```bash
curl -s -X DELETE "http://localhost:8000/v1/user/purge/$USER_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

预期：`{"success":true}`。

---

## 四、验证清理结果

### 步骤 8：确认 User 已删除

```bash
docker exec servora_db psql -U servora -d iam -t -c \
  "SELECT count(*) FROM users WHERE id='$USER_ID';"
```

预期：`0`

### 步骤 9：确认用户拥有的 Organization 已删除

```bash
docker exec servora_db psql -U servora -d iam -t -c \
  "SELECT count(*) FROM organizations WHERE id='$ORG_ID';"
```

预期：`0`

### 步骤 10：确认 Organization 下的 Project 已删除

```bash
docker exec servora_db psql -U servora -d iam -t -c \
  "SELECT count(*) FROM projects WHERE id='$PROJ_ID';"
```

预期：`0`

### 步骤 11：确认 OrganizationMember 已清理

```bash
docker exec servora_db psql -U servora -d iam -t -c \
  "SELECT count(*) FROM organization_members WHERE user_id='$USER_ID';"
```

预期：`0`

### 步骤 12：确认 ProjectMember 已清理

```bash
docker exec servora_db psql -U servora -d iam -t -c \
  "SELECT count(*) FROM project_members WHERE user_id='$USER_ID';"
```

预期：`0`

### 步骤 13：确认 Application 被清理（若有）

```bash
docker exec servora_db psql -U servora -d iam -t -c \
  "SELECT count(*) FROM applications WHERE organization_id='$ORG_ID';"
```

预期：`0`

### 步骤 14：确认执行日志

```bash
docker logs iam 2>&1 | grep "PurgeUser"
```

预期：按顺序看到以下日志：

```
PurgeUser start: user_id=<USER_ID>
PurgeUser PurgeCascade done: user_id=<USER_ID>
PurgeUser FGA cleanup done: user_id=<USER_ID>
PurgeUser Redis cleanup done: user_id=<USER_ID>
PurgeUser complete: user_id=<USER_ID>
```

---

## 五、检查清单

| # | 验证项 | 预期 |
|---|--------|------|
| 1 | PurgeUser 接口返回 | `{"success":true}` |
| 2 | User 行 | 已删除 |
| 3 | 用户拥有的 Organization | 已删除 |
| 4 | Organization 下的 Project | 已删除 |
| 5 | Organization 下的 Application | 已删除 |
| 6 | 用户的 OrganizationMember | 全部清理 |
| 7 | 用户的 ProjectMember | 全部清理 |
| 8 | 执行顺序 | DB → FGA → Redis（通过日志确认） |
| 9 | DB 失败时 FGA/Redis 不执行 | 不在此手动测试覆盖（由单元测试保证） |
| 10 | FGA/Redis 失败时不阻断 | 不在此手动测试覆盖（由单元测试保证） |

---

## 六、单元测试覆盖

上述 #9、#10 由 biz 层单元测试保证，无需启动外部服务：

```bash
go test ./app/iam/service/internal/biz/ -run TestPurge -v
```

覆盖场景：

- `TestPurgeUser_HappyPath` — 正常路径全部成功
- `TestPurgeUser_CascadeFails_StopsEarly` — DB 失败 → 不触发 FGA/Redis
- `TestPurgeUser_FGAFails_StillSucceeds` — FGA 失败不影响整体
- `TestPurgeUser_RedisFails_StillSucceeds` — Redis 失败不影响整体
- `TestPurgeUser_ExecutionOrder_DBBeforeFGABeforeRedis` — 执行顺序验证

---

## 七、常见问题

- **403 AUTHZ_DENIED**：PurgeUser 需要 tenant admin 权限，必须用 admin 用户调用。
- **FGA 相关报错**：确认已执行 `make openfga.init` 且 IAM 容器读取了最新的 `.env`（需 recreate 而非 restart）。
- **keys 为空导致认证失败**：Docker 环境的 `configs/docker/keys/` 需手动放入 `iam.rsa.pem`。
- **Organization 未被删除**：确认该用户在 `organization_members` 中的 `role` 为 `owner`（只有 owner 的 org 会被级联删除）。
