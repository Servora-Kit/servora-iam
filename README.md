# Servora

简体中文

`servora` 是一个微服务快速开发框架，**Proto First** 开发方式，覆盖 API 定义、代码生成、服务实现、前端联调、可观测性与容器化开发链路。

## 核心能力

- **Go workspace + 多模块**：根目录 `go.work` 统一纳管根模块与各服务模块
- **Proto First**：使用 Buf v2 workspace 管理共享 proto 与服务私有 proto
- **双协议接口**：支持 gRPC、HTTP 与 OpenAPI 产物生成
- **DDD 分层**：服务遵循 `service -> biz -> data` 分层
- **依赖注入**：使用 Wire 管理服务依赖图
- **数据访问**：Ent 为主，GORM GEN 作为并行工具链保留
- **可插拔认证**：`pkg/authn` 接口驱动，内置 JWT 引擎与 Keycloak claims 映射
- **细粒度授权**：`pkg/authz` 接口驱动，内置 OpenFGA 引擎
- **全链路审计**：`pkg/audit` 经 Kafka 投递审计事件
- **服务治理**：支持注册发现、配置中心与基础遥测能力
- **可观测性**：集成 OTel / Jaeger / Loki / Prometheus / Grafana

## 技术栈

- 框架：Kratos v2
- API：Protobuf + Buf v2
- DI：Google Wire
- ORM：Ent（主）+ GORM GEN（并行）
- 认证：Keycloak（OIDC）/ JWT / JWKS
- 授权：OpenFGA
- 存储：PostgreSQL + Redis
- 观测：OTel Collector / Jaeger / Loki / Prometheus / Grafana

## 项目结构

```text
.
├── api/                             # 共享 proto 与统一生成产物
│   ├── gen/go/                      # Go 生成代码
│   └── protos/                      # 共享 proto（conf、pagination、authz 注解）
├── app/                             # 服务实现（示例服务）
├── cmd/
│   ├── svr/                         # CLI 工具（svr gen gorm / svr openfga）
│   └── protoc-gen-servora-authz/    # 自定义 protoc 插件
├── pkg/                             # 共享基础库
│   ├── actor/                       # 通用 principal 模型
│   ├── authn/                       # 可插拔认证中间件（JWT / Header / Noop）
│   ├── authz/                       # 可插拔授权中间件（OpenFGA / Noop）
│   ├── audit/                       # 全链路审计
│   ├── bootstrap/                   # 服务启动引导
│   ├── broker/                      # 消息代理抽象（Kafka 实现）
│   ├── db/ent/                      # Ent schema mixin 与 scope 工具
│   ├── governance/                  # 服务治理（注册发现、配置中心）
│   ├── health/                      # 健康检查
│   ├── helpers/                     # 通用工具函数
│   ├── jwt/ & jwks/                 # JWT 签发与 JWKS 验证
│   ├── logger/                      # 日志封装
│   ├── mapper/                      # 对象映射
│   ├── openfga/                     # OpenFGA 客户端封装与缓存
│   ├── redis/                       # Redis 客户端封装
│   └── transport/                   # HTTP/gRPC 传输层工具
├── manifests/                       # 部署清单（k8s / openfga）
├── templates/                       # 通用部署模板
├── docs/                            # 设计文档与参考资料
├── openspec/                        # OpenSpec 变更与归档
├── app.mk                           # 服务级通用 Makefile 模板
├── buf.yaml                         # Buf v2 workspace
├── buf.go.gen.yaml                  # Go 代码生成模板（含 authz + mapper 插件）
├── go.work                          # Go workspace
└── Makefile                         # 根目录统一入口
```

## 快速开始

### 前置要求

- Go 1.26+
- Make
- Docker / Docker Compose

### 克隆仓库

```bash
git clone https://github.com/Servora-Kit/servora.git
cd servora
```

### 配置环境

```bash
cp .env .env.local
# 编辑 .env.local，填入数据库密码、API 密钥等
```

### 安装工具并生成代码

```bash
make init
make gen
```

`make gen` 会统一执行：`api + openapi + wire + ent`。

### 启动容器化开发环境

```bash
# 仅启动基础设施（Consul、Postgres、Redis、OpenFGA、OTel、Jaeger 等）
make compose.up

# 构建开发镜像（首次或 Dockerfile.air / Go 版本变更后需执行，否则 compose.dev 可能因镜像过旧报错）
make compose.dev.build

# 启动基础设施 + Air 热重载开发容器
make compose.dev
```

Compose 管理命令：

```bash
make compose.ps            # 查看基础设施状态
make compose.stop          # 停止基础设施容器
make compose.logs          # 查看基础设施日志
make compose.down          # 移除容器/网络（保留数据卷）
make compose.reset         # 移除容器/网络/数据卷

make compose.dev.ps        # 查看完整开发栈状态
make compose.dev.stop      # 停止完整开发栈容器
make compose.dev.logs      # 查看日志
make compose.dev.restart   # 重启服务
make compose.dev.down      # 移除完整开发栈容器/网络（保留数据卷）
make compose.dev.reset     # 移除完整开发栈容器/网络/数据卷
```

## 开发工作流

1. 修改共享 proto 或服务私有 proto
2. 在仓库根目录执行 `make gen`
3. 在服务目录实现业务代码：`internal/service -> internal/biz -> internal/data`
4. 修改 Wire 依赖图后执行 `make wire`
5. 运行测试与 lint

## 常用命令

```bash
# 代码生成
make gen                    # 统一生成（api + openapi + wire + ent）
make api                    # 仅生成 API（Go + AuthZ）
make openapi                # 仅生成 OpenAPI
make wire                   # 仅生成 Wire
make ent                    # 仅生成 Ent
make build                  # 生成 + 构建所有服务

# 质量检查
make test                   # 运行测试
make cover                  # 测试覆盖率
make lint                   # lint.go + lint.ts（不含 proto；需要时单独 `make lint.proto`）
make lint.go                # Go：根模块 + 各服务子模块（GO_WORKSPACE_MODULES；不含 api/gen 生成代码）
make lint.ts                # TS：`WEB_APPS` 对应 web/* + `api/ts-client`（无对应 script 的包会跳过）
make lint.proto             # Buf proto lint
# 单服务仅扫本模块：cd app/<svc>/service && make lint.go
# 单前端：cd web/<app> && make lint.ts（web-app.mk）

# CLI 工具
svr gen gorm <service...>   # GORM GEN 代码生成
svr openfga init            # 初始化 OpenFGA store
svr openfga model apply     # 更新 OpenFGA model

# OpenFGA 运维
make openfga.init           # 初始化 OpenFGA store
make openfga.model.validate # 验证 model 语法
make openfga.model.test     # 运行 model 测试
make openfga.model.apply    # 应用 model 更新
```

## 可观测性

默认观测组件（Compose 栈）：

- Grafana: `http://localhost:3001`
- Prometheus: `http://localhost:9090`
- Jaeger: `http://localhost:16686`
- Loki: `http://localhost:3100`
- OTel Collector: `4317/4318`

## 质量约束

- 不要手动编辑生成代码：`api/gen/go/`、`wire_gen.go`、`openapi.yaml`、`authz_rules.gen.go`
- 修改 proto 后务必执行 `make gen`
- 修改 Wire 依赖图后务必执行 `make wire`
- 提交前建议通过 `make lint`（或分项 `make lint.go` / `make lint.ts` / `make lint.proto`）与 `make test`（`WEB_APPS` 与 pnpm 一致；`GO_WORKSPACE_MODULES` 为各服务 Go 模块，**不含** `api/gen`）
- 修改 OpenFGA model 后执行 `make openfga.model.apply`

## License

MIT，详见 `LICENSE`。
