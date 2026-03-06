# AGENTS.md - servora 项目根目录

<!-- Generated: 2026-02-09 | Updated: 2026-03-06 -->

## 项目概览

`servora` 是一个基于 Go Kratos v2 的微服务示例仓库，当前采用 **Go workspace + 多模块** 与 **Buf v2 workspace** 组织方式。

当前主线事实：
- 根目录仍保留 `go.mod`，并通过 `go.work` 纳管 `api/gen`、`app/servora/service`、`app/sayhello/service`
- 前端已迁入 `app/servora/service/web/`，不再位于仓库根目录
- Proto 采用三处模块联合编排：`api/protos/`、`app/servora/service/api/protos/`、`app/sayhello/service/api/protos/`
- 共享生成入口在根目录：`make gen`、`make api`、`make openapi`、`make wire`、`make ent`

## 顶层目录

- `api/`：共享 proto、生成产物 `api/gen/` 与相关 AGENTS
- `app/`：服务实现；当前包含 `servora/service/` 与 `sayhello/service/`
- `cmd/svr/`：中心化 CLI，当前提供 `svr gen gorm`
- `pkg/`：共享基础库，现有 `bootstrap`、`governance`、`helpers`、`jwt`、`k8s`、`logger`、`mapper`、`middleware`、`redis`、`transport`
- `manifests/`：统一部署清单，K8s 已收敛到 `manifests/k8s/`
- `docs/`：文档目录；当前包含 `design/`、`knowledge/`、`reference/`
- `openspec/`：OpenSpec 变更与归档

## 关键文件

- `Makefile`：根构建入口，负责 `api`、`openapi`、`wire`、`ent`、构建与 Compose
- `app.mk`：服务级通用 Makefile；服务目录中的 `Makefile` 通过 `include ../../../app.mk` 复用
- `buf.yaml`：Buf v2 workspace，声明三个 proto module 路径
- `buf.go.gen.yaml`：根级 Go 代码生成模板，输出到 `api/gen/go`
- `go.work` / `go.work.sum`：多模块工作区配置
- `README.md`：项目入口说明

## 当前目录约定

### API / Proto
- 共享 proto 放在 `api/protos/`
- `servora` 服务 proto 放在 `app/servora/service/api/protos/`
- `sayhello` 服务 proto 放在 `app/sayhello/service/api/protos/`
- Go 生成代码统一输出到 `api/gen/go/`

### 服务实现
- `app/servora/service/`：主服务，包含 `api/`、`cmd/`、`internal/`、`configs/`、`web/`
- `app/sayhello/service/`：独立示例服务，包含自己的 `api/` 与运行时目录

### 前端
- 目录：`app/servora/service/web/`
- 生成的 TypeScript HTTP 客户端输出到 `app/servora/service/web/src/service/gen/`

### 部署
- K8s 基础设施：`manifests/k8s/base/`
- 服务清单：`manifests/k8s/servora/`、`manifests/k8s/sayhello/`

## 常用命令

在项目根目录执行：

```bash
make init
make gen
make api
make openapi
make wire
make ent
make build
make test
make lint.go
make compose.build
make compose.up
make compose.dev
make compose.dev.up
make compose.dev.restart
```

CLI：

```bash
svr new api <name> <server_name>
svr new api billing servora
svr new api billing.invoice servora
svr gen gorm <service-name...>
svr gen gorm servora --dry-run
```

## 维护提示

- 根 `make api` 当前固定使用 `buf.go.gen.yaml`；TypeScript 生成由服务目录内的 `api/buf.typescript.gen.yaml` 单独驱动
- 修改任意 proto 后优先执行根目录 `make gen`
- 修改服务依赖注入后执行对应服务目录下的 `make wire`
- 不要手改 `api/gen/go/`、`wire_gen.go`、`openapi.yaml`
- 若文档涉及前端路径，统一使用 `app/servora/service/web/`
