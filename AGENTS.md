# AGENTS.md - servora-iam

<!-- Generated: 2026-03-25 | Updated: 2026-03-25 -->

## 项目概览

`servora-iam` 是 [Servora](https://github.com/Servora-Kit/servora) 框架的**示例项目**，包含 IAM（身份与访问管理）微服务、SayHello 示例服务及 IAM 前端应用。

依赖关系：
- Go module 依赖：`github.com/Servora-Kit/servora`、`github.com/Servora-Kit/servora/api/gen`
- Proto BSR 依赖：`buf.build/servora/servora`
- 业务 proto 发布到：`buf.build/servora/servora-iam`
- Go module 路径：
  - `github.com/Servora-Kit/servora-iam/app/iam/service`
  - `github.com/Servora-Kit/servora-iam/app/sayhello/service`
  - `github.com/Servora-Kit/servora-iam/api/gen`

当前主线事实：
- 所有开发在 `main` 分支进行
- `go.work` 已 gitignore，仅用于仓库内部多模块联合与顶层跨仓库开发
- 前端采用 pnpm workspace，包含 `web/iam/`、`web/pkg/`、`web/ui/`

## 开发约束

### 提交消息格式

遵循 Servora-Kit 组织统一规范：

```
type(scope): description
```

**允许的 type**：`feat`、`fix`、`refactor`、`docs`、`test`、`chore`

**建议的 scope**：
- `api`：API / Proto / OpenAPI
- `app/iam`：IAM 服务
- `app/sayhello`：SayHello 服务
- `web`：前端应用
- `web/pkg`：前端共享工具库
- `web/ui`：前端共享 UI 组件库
- `manifests`：部署清单
- `infra`：基础设施/部署
- `repo`：仓库治理

## 顶层目录

- `api/`：生成代码产物
  - `gen/go/`：Go 生成代码（业务 proto）
  - `gen/ts/`：TypeScript 生成代码
  - `ts-client/`：pnpm workspace 包锚点（`@servora/api-client`），引用 `gen/ts/` 中的生成代码
- `app/`：微服务实现
  - `iam/service/`：IAM 微服务（认证、授权、组织、项目）
    - `api/protos/`：业务 proto（authn、authz、iam、user 等）
    - `cmd/`：服务入口
    - `configs/`：配置文件
    - `internal/`：业务实现（service/biz/data/server/middleware）
  - `sayhello/service/`：SayHello 示例服务
    - `api/protos/`：业务 proto
    - `cmd/`、`configs/`、`internal/`
- `web/`：前端工作区（pnpm workspace）
  - `iam/`：IAM 前端应用
  - `pkg/`：共享前端工具库（`@servora/web-pkg`）：请求处理、Token 管理、Kratos 错误解析
  - `ui/`：共享 UI 组件库（`@servora/ui`）
- `manifests/`：部署清单
  - `k8s/iam/`、`k8s/sayhello/`：K8s 部署
  - `openfga/`：OpenFGA model 与测试

## 关键文件

- `Makefile`：构建入口（gen / api / wire / ent / lint / test / compose / pnpm / openfga）
- `buf.yaml`：Buf v2 workspace，包含 `app/iam/service/api/protos`（名为 `buf.build/servora/servora-iam`）和 `app/sayhello/service/api/protos`；依赖 `buf.build/servora/servora`
- `buf.go.gen.yaml`：Go 代码生成模板（含 servora 自定义插件：authz、mapper、audit）
- `buf.typescript.gen.yaml`：TS 代码生成模板
- `docker-compose.yaml`：基础设施（consul、db、redis、openfga 等）
- `docker-compose.dev.yaml`：开发环境（iam + sayhello 服务）
- `pnpm-workspace.yaml`：前端 monorepo 配置
- `.env.example`：环境变量模板

## 目录约定

### API / Proto
- IAM 业务 proto：`app/iam/service/api/protos/`
- SayHello 业务 proto：`app/sayhello/service/api/protos/`
- 框架公共 proto 通过 BSR 依赖（`buf.build/servora/servora`），不在本仓库存放
- Go 生成代码输出到 `api/gen/go/`
- TS 生成代码输出到 `api/gen/ts/`

### Proto 命名规范
- `package` 以 `servora.` 开头，携带版本后缀
- 目录与 `package` 逐段对齐（Buf `PACKAGE_DIRECTORY_MATCH`）
- `go_package` 落到 `github.com/Servora-Kit/servora-iam/api/gen/go/servora/**`

### 服务实现
- DDD 分层：`service -> biz -> data`
- 认证/授权中间件位于 `app/iam/service/internal/server/middleware/`
- Wire 依赖注入：修改后执行 `make wire`

### 前端
- 前端应用：`web/iam/`
- 通过 `@servora/api-client/<namespace>/...` 引用 TS 生成类型
- 通过 `@servora/web-pkg/<module>` 引用共享工具
- 通过 `@servora/ui` 引用共享 UI 组件

## 常用命令

```bash
# 初始化
make init              # 安装工具（protoc 插件 + CLI + 前端依赖）

# 代码生成
make gen               # 统一生成（api + wire + ent + openapi + ts）
make api               # 仅生成 proto 代码
make wire              # 仅生成 Wire
make ent               # 仅生成 Ent

# 质量检查
make test              # 运行测试
make lint              # lint.go + lint.ts
make lint.proto        # Proto lint

# 前端
make pnpm.install      # 安装前端依赖
make dev.web           # 启动前端开发服务器

# Compose
make compose.up        # 启动基础设施
make compose.dev       # 启动开发环境
make compose.stop      # 停止
make compose.down      # 移除容器/网络
make compose.reset     # 移除容器/网络/数据卷

# OpenFGA
make openfga.init             # 初始化 store
make openfga.model.validate   # 验证 model
make openfga.model.test       # 测试 model
make openfga.model.apply      # 应用 model 更新
```

## 维护提示

- 修改 proto 后执行 `make gen`
- 修改 Wire 依赖图后执行 `make wire`
- 不要手改 `api/gen/go/`、`api/gen/ts/`、`wire_gen.go`、`openapi.yaml`、`*_rules.gen.go`
- `api/ts-client/` 仅为 pnpm workspace 包锚点，不放自定义代码
- `web/pkg/` 放通用逻辑，不放业务代码
- 修改 OpenFGA model 后执行 `make openfga.model.apply`
- 自定义 protoc 插件通过 `go install github.com/Servora-Kit/servora/cmd/...@latest` 安装
