# servora TODO

## 技术债务

- [x] **gRPC 客户端配置查找优化** - `pkg/transport/client/grpc_conn.go` 改为使用 `NewClient` 启动阶段构建的 `service_name -> config` 索引，避免热路径线性扫描

## Oracle 审查记录（2026-02-26）

- [x] **Tracing 明文传输 + 全采样** - `pkg/governance/telemetry/tracing.go` 改为显式配置 `insecure` / `sampling_ratio` / `ca_path`，并采用更安全的默认采样策略
- [x] **Collector debug exporter 在生产链路** - `manifests/otel/otel-collector.yaml` 与模板已移除 traces/logs pipeline 中的 `debug`
- [x] **Collector 未纳入健康依赖链** - `docker-compose.yaml` 为 `otel-collector` 增加健康检查，`docker-compose.dev.yaml` 改为 `service_healthy`
- [ ] **sayhello metrics 可观测性闭环不完整** - 暂缓：按当前决策，`sayhello` 暂时不暴露指标路由，Prometheus 仍仅抓取 `servora`
- [x] **Prometheus 抓取范围偏窄** - 已补充 `otel-collector`、`loki`、`jaeger`、`grafana` 的组件级采集

## devops相关

- [x] make compose.logs会显示所有日志，但是compose.dev.logs会排除掉所有基础设施的日志只显示微服务日志
