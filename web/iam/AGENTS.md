# AGENTS.md - web/iam

<!-- Parent: ../../AGENTS.md -->
<!-- Generated: 2026-03-15 | Updated: 2026-03-17 -->

## 目录定位

`web/iam` 是 IAM 的前端应用（Vite + React + TanStack Router + TanStack Query），与 `app/iam/service` 后端对接。
属于根目录 pnpm workspace（`pnpm-workspace.yaml`）的成员之一。

## 使用生成类型

**不要手写请求/响应类型，优先使用 Proto 生成的 TypeScript 类型。**

所有生成代码由 `make api-ts` 输出到 `api/gen/ts/`，通过 pnpm workspace 包 `@servora/api-client` 引用：

```ts
// IAM 服务 HTTP 客户端与类型
import { createIamServiceClient } from '@servora/api-client/iam/service/v1/index'

// 认证 / 用户 / 组织 / 项目
import type { ... } from '@servora/api-client/authn/service/v1/index'
import type { ... } from '@servora/api-client/user/service/v1/index'
import type { ... } from '@servora/api-client/organization/service/v1/index'
import type { ... } from '@servora/api-client/project/service/v1/index'

// 公共分页类型
import type { PaginationRequest } from '@servora/api-client/pagination/v1/index'
```

发请求使用单例 `iamClients`（`#/api`），见 `src/api.ts` 与 `src/service/request/`。

## 常用命令

```bash
# 在仓库根目录执行
make api-ts          # 重新生成 TypeScript 客户端（修改 proto 后必须执行）
pnpm install         # 安装/更新 workspace 依赖

# 在 web/iam/ 目录执行
pnpm dev             # 启动开发服务器（端口 3000）
pnpm build           # 生产构建
pnpm test            # 运行测试
pnpm check           # prettier + eslint 格式化与修复
```

## 新增前端应用

在 `web/<service>/` 下新建应用时，按以下步骤接入 `@servora/api-client`：

**1. `package.json` 添加 workspace 依赖**
```json
{
  "dependencies": {
    "@servora/api-client": "workspace:*"
  }
}
```

**2. `tsconfig.json` 添加路径别名**
```json
{
  "compilerOptions": {
    "paths": {
      "@servora/api-client/*": ["../../api/ts-client/*", "../../api/gen/ts/*"]
    }
  }
}
```

**3. 在服务的 `api/buf.typescript.gen.yaml` 设置 `clean: false`，输出到 `api/gen/ts/`**
```yaml
clean: false
plugins:
  - local: protoc-gen-typescript-http
    out: api/gen/ts
```

**4. 运行 `pnpm install` 链接 workspace**

之后直接 `import from '@servora/api-client/<namespace>/...'` 即可。
