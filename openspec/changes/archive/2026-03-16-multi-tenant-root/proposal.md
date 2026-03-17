## Why

servora 当前以单个硬编码 Tenant（slug=root）运行，所有 Organization 都挂在同一 tenant 下。多租户前置工作（数据层 scope、FGA 回滚、Ent 查询规范、PurgeUser 级联）已全部完成。

要支持 SaaS 级别的多租户（多个独立客户/租户），需要将 Tenant 从「配置常量」升级为「可 CRUD 的一等实体」，并完善 Tenant 成员管理、scope 校验与 FGA 一致性。

参照 Kemate 的 B2B2C 多租户设计（Tenant → Workspace → Application 三级结构），映射到 servora 为 Tenant → Organization → Project。同时引入 Platform 概念（FGA type）区分平台管理与租户管理。

## What Changes

- **新增**：Tenant 成为可 CRUD 的一等实体（Ent schema、biz、data、service 全栈）
- **新增**：TenantMember 实体，含角色（owner/admin/member）+ 状态（active/invited）
- **新增**：CreateTenant 一体化流程（创建租户 + 默认组织 + 默认项目 + FGA 同步）
- **新增**：Personal Tenant，用户注册时自动创建，支持 B2C 场景
- **新增**：邀请流程，三层 Member 统一 status 字段
- **新增**：FGA `platform` type，区分平台管理 vs 租户管理
- **修改**：Organization 查询按 tenant_id 过滤（defense-in-depth）
- **修改**：去除 TenantRootID 硬编码依赖，改为从请求上下文动态获取
- **修改**：FGA 模型扩展 tenant owner、organization 权限从 tenant 继承

## Capabilities

### New Capabilities

- `tenant-entity`：Tenant + TenantMember 的 Ent schema、biz、data、service 全栈
- `create-tenant-flow`：CreateTenant → 默认 Organization → 默认 Project → FGA 一体化
- `personal-tenant`：注册时自动创建 personal tenant（EnsurePersonalTenant）
- `invitation-flow`：Member 实体统一 status 字段（active/invited），支持邀请→接受→拒绝流程

### Modified Capabilities

- `tenant-scope`：Organization 按 tenant_id 过滤、请求级 tenant 上下文（X-Tenant-ID middleware）
- `fga-model-extension`：引入 platform FGA type、tenant owner + organization 权限继承

## Impact

- **代码**：大量新增文件（schema、entity、biz、data、service、proto），修改 organization/user/project biz 层和 authz 中间件
- **API**：新增 Tenant CRUD + Member 管理 API；修改 Organization API 签名（tenantID 参数化）；BREAKING：去除 TenantRootID
- **FGA**：新增 `platform` type，扩展 `tenant` type，organization 增加 `from tenant` 继承
- **依赖**：无新增外部依赖
