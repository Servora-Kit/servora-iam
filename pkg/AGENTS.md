# AGENTS.md - pkg 共享库层

<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-02-09 | Updated: 2026-03-06 -->

## 目录概览

`pkg/` 提供跨服务复用的基础能力。当前真实子目录如下：

```text
pkg/
├── bootstrap/
├── governance/
├── helpers/
├── jwt/
├── k8s/
├── logger/
├── mapper/
├── middleware/
├── redis/
└── transport/
```

## 模块速览

- `bootstrap/`：启动链路与配置加载复用
- `governance/`：注册发现、配置中心、遥测
- `helpers/`：通用辅助函数与 bcrypt 哈希
- `jwt/`：泛型 JWT 工具与 context 注入
- `k8s/`：Kubernetes 客户端工具
- `logger/`：Kratos + Zap 日志适配，含 GORM / Ent 日志桥接
- `mapper/`：模型映射工具
- `middleware/`：公共中间件，如 CORS 与 operation 白名单
- `redis/`：Redis 客户端封装
- `transport/`：服务间 transport client

## 当前事实

- `governance/` 下已经是 `config/`、`registry/`、`telemetry/`，旧的 `configCenter/` 描述已失效
- `logger/` 除 `gorm_log.go` 外还有 `ent_log.go`
- `middleware/whitelist.go` 实现的是 **operation 白名单**，不是 IP 白名单
- `redis/` 当前目录没有单独测试文件

## 开发约定

- 优先保持无状态、可复用、低耦合
- 需要资源释放时返回 `cleanup func()`
- 不在库代码里 `panic` 或 `log.Fatal`
- 依赖生成配置类型时，从 `api/gen/go/conf/v1` 导入

## 常用命令

```bash
go test ./pkg/...
go test ./pkg/logger/...
go test ./pkg/governance/registry/...
go test ./pkg/redis/...
```

## 维护提示

- 若更新共享基础设施能力，优先同步 `pkg/AGENTS.md` 与对应子模块 `AGENTS.md`
- `helpers` 承担密码哈希职责，旧的 `pkg/hash` 说法已经不适用
