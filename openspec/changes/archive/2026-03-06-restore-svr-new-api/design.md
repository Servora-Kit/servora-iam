## 上下文

当前仓库的 proto 已经拆成三块：共享 proto 在 `api/protos/`，服务私有 proto 分别位于 `app/servora/service/api/protos/` 和 `app/sayhello/service/api/protos/`。`svr` CLI 目前只保留 `svr gen gorm`，因此新建 gRPC 协议时只能手工创建目录、主 proto 和 doc proto，目录命名与服务归属都靠人工维护。

这次恢复的不是旧版“写入共享 proto 目录”的 `svr new api`，而是一个服务维度的脚手架命令：`svr new api <name> <server_name>`。它必须与当前 Buf v2 workspace、服务目录约定、以及服务自己的 OpenAPI / TypeScript 生成方式兼容，同时避免重新引入 HTTP `i_*.proto` 这类与业务风格强绑定的模板判断。

## 目标 / 非目标

**目标：**
- 恢复 `svr new api <name> <server_name>` 命令，并挂回 `svr new` 命令组
- 目标输出固定为 `app/<server_name>/service/api/protos/<name>/service/v1/`
- 只生成 gRPC proto 与对应 doc proto，不生成 HTTP 专用 `i_*.proto`
- 复用现有服务发现/校验能力，确保 `server_name` 必须对应一个真实服务目录
- 保持 `make gen` / `make api` / 服务级 OpenAPI 与 TS 生成流程不需要额外配置迁移

**非目标：**
- 不自动修改服务级 `buf.typescript.gen.yaml` 或 `buf.openapi.gen.yaml` 的 `inputs.paths`
- 不自动运行 `buf generate`、`make gen` 或 `make openapi`
- 不生成 HTTP/BFF 风格接口 proto
- 不处理“新建服务”场景；命令只面向已存在的 `app/<service>/service`

## 决策

### D1：命令签名改为显式服务维度

命令使用：

```text
svr new api <name> <server_name>
```

其中 `<name>` 表示 proto 模块名，`<server_name>` 表示目标服务目录名。

**为什么选这个方案：**
- 与当前服务私有 proto 布局直接对齐，避免继续写入 `api/protos/`
- 比从 cwd 猜服务、或通过交互选择服务更明确，CI 也更稳定

**为什么不选这些方案：**
- `svr new api <name>`：无法判断输出到哪个服务目录
- `svr new api <name> --service <server_name>`：可行，但对当前 CLI 来说不如第二位置参数直接

### D2：只支持通用 gRPC proto 模板

输出文件固定为：
- `<name>.proto`
- `<name>_doc.proto`

不生成 `i_<name>.proto`。

**为什么选这个方案：**
- 当前仓库服务间 proto 风格不完全对称，`servora` 有 `i_*.proto`，`sayhello` 没有
- 先收敛到“通用 gRPC proto 脚手架”，可以避免脚手架反过来强行规定服务内部架构

**为什么不选这些方案：**
- 自动按服务生成不同模板：会把服务内部实现风格编码进 CLI，维护成本高
- 引入 `--kind grpc|http`：未来可能有价值，但当前用户已明确“不用生成 http 代码”

### D3：目标目录按服务内 proto 约定推导

目录规则：

```text
app/<server_name>/service/api/protos/<name>/service/v1/
```

示例：

```text
svr new api billing servora
-> app/servora/service/api/protos/billing/service/v1/

svr new api inventory sayhello
-> app/sayhello/service/api/protos/inventory/service/v1/
```

**为什么选这个方案：**
- 与 `buf.yaml` 中的服务模块路径天然兼容，无需新增 workspace module
- 与现有服务目录结构一致，开发者能在服务上下文里继续工作

**为什么不选这些方案：**
- 继续写入 `api/protos/<name>/service/v1/`：与当前仓库主线相矛盾
- 直接写入 `app/<server_name>/service/api/protos/<server_name>/service/v1/`：会把“服务名”和“业务模块名”混为一谈，无法支持一个服务下多个业务 proto 模块

### D4：服务名校验复用现有 discovery 能力

命令应复用 `cmd/svr/internal/discovery/config.go` 中已有的服务目录扫描与校验逻辑，至少保证：
- `server_name` 对应 `app/<server_name>/service`
- 不存在时返回可读错误，并列出可用服务

**为什么选这个方案：**
- 已有实现已经被 `svr gen gorm` 使用，行为可复用
- 可保持 CLI 错误语义一致

### D5：模板源恢复为 CLI 自带模板

恢复命令时，需要重新拥有受版本控制的模板源，但模板应绑定“服务内 gRPC proto”语义，而不是旧的共享 proto 语义。

推荐保留两层：
- CLI 内置模板（用于命令兜底）
- 仓库内可读模板目录（便于项目内查看与调整）

**为什么选这个方案：**
- 既保证命令在任何环境可运行，也保留项目内可见的模板基准

**权衡：**
- 会重新引入模板同步问题，但比把模板硬编码进 Go 字符串更可维护

## 风险 / 权衡

- 服务配置不对称 → 脚手架只生成 proto 文件，不自动改 OpenAPI / TS 配置
- 新建模块可能不会立即进入服务级生成模板 → 通过文档明确“生成后若需 OpenAPI / TS，开发者自行补充 service api 模板路径”
- 模板恢复后会重新引入文档与规范维护面 → 将范围限定为 gRPC proto，避免一开始支持过多变体
- 旧 `svr new api` 已被下线 → 恢复时必须同步 README、AGENTS、OpenSpec，避免命令重新出现但文档仍写“当前只保留 svr gen gorm”

## 迁移计划

1. 恢复 `svr new api` 的命令注册与实现，但采用新签名
2. 恢复并调整模板文件，使其面向服务内 gRPC proto 输出
3. 更新现行 README / AGENTS / OpenSpec
4. 通过 `go run ./cmd/svr --help`、`make api` 与目标路径生成验证命令恢复后的行为

该变更属于功能恢复 + 语义重定义，不涉及生产数据迁移。

## 开放问题

- 是否在首次实现中支持点分层级名称（如 `billing.invoice`）保持多级目录映射？当前建议保留，以延续旧命令的命名能力。
- 是否在 `svr new api` 成功后提示开发者检查服务级 `api/buf.openapi.gen.yaml` / `api/buf.typescript.gen.yaml` 的 `inputs.paths`？当前建议只提示，不自动修改。
