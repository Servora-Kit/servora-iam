# Spec: logger-refactor

> 对应 design.md D0

## 需求

### R1: 创建 API 简化

- `logger.New(app *conf.App) *ZapLogger` 直接接收 proto 配置创建 logger
- `New(nil)` 不 panic，返回 dev 模式 console logger（sensible defaults）
- 返回类型为 `*ZapLogger`（而非 `log.Logger`），便于调用 `Zap()`/`Sync()`
- 移除 `logger.Config` struct 和 `NewLogger` 函数

### R2: For 快捷方法

- `logger.For(l log.Logger, module string) *Helper` 一行创建带 module 的 helper
- 等价于 `logger.NewHelper(l, logger.WithModule(module))`
- 所有现有 `NewHelper(l, WithModule("..."))` 调用全量替换为 `For`

### R3: With 简化

- `logger.With(l log.Logger, module string) log.Logger` 支持直接传 module string
- 兼容 `logger.With(l, WithModule("..."), WithField("..."))` 的 Option 风格
- 所有现有 `With(l, WithModule("..."))` 调用迁移为 `With(l, "module")`

### R4: Zap() getter

- `*ZapLogger` 新增 `Zap() *zap.Logger` 公开方法，暴露底层 zap 实例
- GORM bridge (`GetGormLogger`)、Ent bridge (`EntLogFuncFrom`) 内部使用 `Zap()` 替代字段直接访问
- 下游 `pkg/broker`（franz-go kzap 插件）可通过此方法获取 `*zap.Logger`

### R5: Sync 字段 → 方法

- 移除 `Sync func() error` exported 字段
- 新增 `Sync() error` 方法，委托至 `l.log.Sync()`
- 无调用方需迁移（字段从未被外部使用）

### R6: 内部实现优化

- 提取 `buildCore(env, level, filename string, ...) zapcore.Core` 消除 prod/default 20 行重复
- test 环境使用 `zap.NewNop()` 或极简 core

### R7: 调用方全量迁移

- `pkg/bootstrap/bootstrap.go`: `NewLogger(&Config{...})` → `New(bc.App)`
- `app/iam/service/**/*.go`: 所有 `NewHelper(l, WithModule("..."))` → `For(l, "...")`
- `app/sayhello/service/**/*.go`: 同上
- `pkg/redis/redis.go`, `pkg/openfga/config.go`, `pkg/transport/**/*.go`, `pkg/jwks/endpoints.go`, `pkg/governance/telemetry/metrics.go`: 同上
- module 命名去掉 `-service` 后缀（如 `"user/data/iam-service"` → `"user/data/iam"`）

### R8: 测试更新

- 现有测试（`log_defaults_test.go`、`gorm_log_test.go`）适配新 API
- 新增 `New(nil)` 安全性测试
- 新增 `For` 方法测试
- 新增 `Zap()` getter 测试
- 新增 `Sync()` 方法测试

## 场景

### S1: Bootstrap 创建 logger（最简路径）

```go
appLogger := logger.New(bc.App)
// appLogger 是 *ZapLogger，满足 log.Logger 接口
// 可以直接 appLogger.Zap() 获取 *zap.Logger
// 可以直接 defer appLogger.Sync()
```

### S2: 业务层创建模块 Helper

```go
type UserBiz struct {
    log *log.Helper
}

func NewUserBiz(l log.Logger) *UserBiz {
    return &UserBiz{
        log: logger.For(l, "user/biz/iam"),
    }
}
```

### S3: 中间件获取带 module 的 logger

```go
ms := svrmw.NewChainBuilder(logger.With(l, "http/server/iam")).
    Use(recovery.Recovery()).
    Build()
```

### S4: GORM/Ent bridge 使用 Zap getter

```go
gormLog := logger.GormLoggerFrom(l, "gorm/data/iam")
entLog := logger.EntLogFuncFrom(l, "ent/data/iam")
```

### S5: franz-go kzap 集成

```go
zapLogger, ok := l.(*logger.ZapLogger)
if ok {
    kzapOpt := kzap.New(zapLogger.Zap())
    client, _ = kgo.NewClient(kzap.Opt(kzapOpt))
}
```

### S6: nil config 安全

```go
l := logger.New(nil)
// 不 panic，返回 dev 模式 console logger
l.Log(log.LevelInfo, "msg", "works with nil config")
```
