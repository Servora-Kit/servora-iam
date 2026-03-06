## 1. 恢复 CLI 命令骨架

- [x] 1.1 恢复 `cmd/svr/internal/cmd/new/` 命令组与根命令注册，但将 `api` 子命令签名改为 `svr new api <name> <server_name>`
- [x] 1.2 复用 `cmd/svr/internal/discovery/` 的服务扫描/校验逻辑，确保 `server_name` 必须映射到真实的 `app/<service>/service` 目录
- [x] 1.3 保留 `<name>` 的 snake_case / 点分层级校验，并将目标目录改为 `app/<server_name>/service/api/protos/<name>/service/v1/`

## 2. 恢复并调整 proto 模板

- [x] 2.1 恢复 CLI 使用的 proto 模板与 doc 模板，并将默认输出语义调整为服务内 gRPC proto
- [x] 2.2 确认模板只生成 `<name>.proto` 与 `<name>_doc.proto`，不生成 `i_*.proto`
- [x] 2.3 若保留仓库内可见模板目录，确保其路径与 README / AGENTS / OpenSpec 的现行描述一致

## 3. 同步文档与规范

- [x] 3.1 更新 `README.md`、`AGENTS.md`、`cmd/svr/AGENTS.md`、`api/AGENTS.md`、`api/protos/AGENTS.md`，重新说明 `svr new api <name> <server_name>` 的用途与限制
- [x] 3.2 更新现行 OpenSpec 说明，使 CLI 能力与服务内 proto 组织约定一致
- [x] 3.3 明确文档提示：生成完成后如需进入 OpenAPI / TypeScript 生成链路，开发者需要自行检查服务级 `api/buf.openapi.gen.yaml` 或 `api/buf.typescript.gen.yaml`

## 4. 验证

- [x] 4.1 运行 `go run ./cmd/svr --help` 与 `go run ./cmd/svr new --help`，确认命令重新暴露且帮助信息正确
- [x] 4.2 运行 `go run ./cmd/svr new api billing servora`，验证生成 `app/servora/service/api/protos/billing/service/v1/billing.proto` 与 `billing_doc.proto`
- [x] 4.3 运行 `go run ./cmd/svr new api billing.invoice servora`，验证点分层级映射到多级目录且文件名转为下划线拼接
- [x] 4.4 运行 `go run ./cmd/svr new api billing notfound` 与非法名称输入，验证错误提示与非零退出码
- [x] 4.5 运行 `make gen` 与 `make api`，确认现有 Buf 生成流程不因命令恢复而中断
