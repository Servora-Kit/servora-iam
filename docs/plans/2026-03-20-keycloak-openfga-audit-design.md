# 设计文档：Servora 接入 Keycloak 后的认证、授权、审计与框架演进

**日期：** 2026-03-20
**状态：** 草案

## 1. 背景与目标

Servora 当前仓库同时承载了框架能力（`pkg/`、`cmd/`、`api/`）与一个早期自建 IAM 服务实现。现阶段已经明确：

- 未来希望将 **Servora 打造成面向微服务快速开发的脚手架与框架生态**；
- `pkg/` 中的能力会逐步框架化、通用化，并最终作为 **Servora 生态 Go 包** 对外发布；
- 当前 IAM 中的 `user / org / project / application` 等对象，大量属于早期验证 IAM 能力的测试性产物，并不是已落地业务领域模型；
- 后续不仅审计会用到消息队列，框架层还会逐步支持更通用的 broker / transport / queue 能力；
- 认证希望引入 **Keycloak**，授权继续采用 **OpenFGA**，并为后续 **Audit Service + Kafka** 做统一设计。

本文目标是给出一份以**设计决策**为主、兼顾 **Servora 框架演进方向**的文档，明确：

1. 接入 Keycloak 后，网关、业务服务、OpenFGA、Audit Service 各自负责什么；
2. 是否还需要保留“中央 IAM/AuthZ 在线代理服务”；
3. 审计体系如何落地，以及如何与 proto / codegen / middleware 深度结合；
4. 当前 Servora 中什么该删、什么该保留、什么该替换；
5. Servora 未来的 broker / transport / audit / codegen 生态应该朝什么方向演进。

---

## 2. 当前架构与现状判断

### 2.1 当前认证链路

当前 Servora 的 IAM 服务本身兼任了一部分 issuer 能力：

- 暴露 `/.well-known/jwks.json`
- 暴露 `/.well-known/openid-configuration`
- 提供登录、注册、刷新 token、邮箱验证、密码重置等认证流程
- 通过 `GET /v1/auth/verify` 供 Traefik ForwardAuth 调用

现有主链路并不是“所有业务服务自己拉 JWKS 验 token”，而是：

```text
Client -> Traefik -> IAM /v1/auth/verify -> 注入 X-User-ID -> 上游服务
```

其中：

- Traefik 负责统一入口与 ForwardAuth 调用；
- IAM 负责校验 `Authorization`；
- 上游服务通过网关注入的 `X-User-ID` 构建 actor；
- `pkg/transport/server/middleware/identity.go` 已经体现了“信任网关身份头”的模式雏形。

### 2.2 当前授权链路

当前 `pkg/authz` 已经具备比较理想的通用授权执行能力：

- 基于 operation 查找规则；
- 从 context 中读取 actor；
- 从 proto request 中提取 object id；
- 调用 OpenFGA `Check` / `CachedCheck`；
- 在授权失败时统一返回错误。

这意味着，**Servora 已经拥有“各业务服务本地接入统一 authz middleware”的核心基础**。

### 2.3 当前问题

当前实现虽然可运行，但与未来目标存在明显偏差：

1. **认证中心与框架能力耦合过深**
   自建 IAM 同时承担登录、签发 JWT、暴露 JWKS、ForwardAuth verify、OpenFGA 运维等多重职责，边界不清。

2. **中央 IAM 容易继续膨胀成在线认证/授权代理**
   若继续强化 IAM，对下游服务会形成新的中心依赖。

3. **审计体系尚未形成统一骨架**
   还没有基于统一 event schema、统一 middleware、统一消息总线的审计架构。

4. **broker / transport / audit 还未进入框架级设计**
   现在更多是服务内实现，尚未形成 Servora `pkg/` 生态。

---

## 3. 核心决策

| 决策点 | 结论 |
|---|---|
| 认证中心 | 使用 **Keycloak** |
| 网关认证策略 | 由 **Traefik / Gateway 统一验 token** |
| 业务服务是否重复验 JWT | 默认 **不重复验**，优先信任网关注入的 principal header |
| 授权底座 | 继续使用 **OpenFGA** |
| 授权执行位置 | **各业务服务本地** 接入 `pkg/authz` |
| 是否保留中央 IAM/AuthZ 在线代理 | **不保留** 为主；最多保留薄的管理/后台能力 |
| 审计架构 | **中心化 Audit Service + 非中心化 authz/audit emit** |
| 审计总线 | 先支持 **Kafka**，后续框架化支持更多 broker |
| actor 模型 | 设计为 **通用 principal 模型**，而不是直接镜像 Keycloak claims |
| 审计规则配置方式 | 采用 **all-in-proto + 注解 + 代码生成 + middleware** |
| broker / transport 演进方向 | 在 Servora 内部建设自有 `pkg` 生态，参考外部项目但不以其为核心依赖 |

---

## 4. 接入 Keycloak 后的职责分工

### 4.1 Keycloak

Keycloak 负责：

- 用户认证；
- OIDC / OAuth2 标准流程；
- token 签发；
- JWKS / discovery；
- client / realm / role 等身份提供方维度的管理能力。

Keycloak 不负责：

- 业务资源级授权；
- OpenFGA 关系建模；
- 业务审计中心存储；
- 作为所有服务的在线授权代理。

### 4.2 网关（Traefik）

网关负责：

- 统一入口；
- 对接 Keycloak 认证链路；
- 验证 token；
- 将 principal 信息注入上游请求头；
- 做粗粒度入口控制；
- 记录认证边界层审计（如认证成功/失败、路由到哪个上游）。

网关不负责：

- 细粒度授权判断；
- 解析业务资源 ID；
- 记录业务资源变更审计；
- 调用 OpenFGA 做通用 check。

### 4.3 各业务服务

各业务服务负责：

- 从 gateway 注入的 identity header 构建 actor / principal；
- 本地执行 `pkg/authz`；
- 直接调用 OpenFGA；
- 产出授权决策审计、关系变更审计、资源变更审计；
- 执行业务逻辑。

各业务服务不负责：

- 自建认证中心；
- 自己实现完整登录 / refresh / discovery / JWKS 主链；
- 搭建中央 authz proxy。

### 4.4 OpenFGA

OpenFGA 负责：

- 关系模型存储；
- `Check` / `ListObjects` / tuple write / tuple delete；
- 为业务服务提供统一授权基础设施。

OpenFGA 不负责：

- token 验证；
- 网关鉴权；
- 审计查询服务；
- 业务 API 代理。

### 4.5 Audit Service

Audit Service 负责：

- 消费审计 topic；
- 校验与反序列化审计事件；
- 落库（推荐 ClickHouse）；
- 提供查询、统计、筛选能力。

Audit Service 不负责：

- 在线鉴权代理；
- 在线认证；
- 直接参与业务请求主链路。

---

## 5. 为什么不保留中央 IAM/AuthZ 在线代理

### 5.1 原则

本次设计推荐：

- **中心化审计**；
- **非中心化授权执行**；
- **认证中心交给 Keycloak**；
- **授权执行内嵌到业务服务**。

### 5.2 不保留中央在线 authz 代理的原因

1. **现有 `pkg/authz` 已具备通用执行能力**
   没有必要再对 OpenFGA 套一层 HTTP / gRPC 代理服务。

2. **OpenFGA 自身已经是独立基础设施**
   它本身具备服务化与 HA 能力，不需要人为再加一层中心代理。

3. **减少网络跳数与故障面**
   业务服务直接调用 OpenFGA，比“业务服务 -> IAM/AuthZ Service -> OpenFGA”更简单、更可控。

4. **更符合微服务边界**
   授权决策发生在业务服务的 operation 上，本地执行更容易获得业务上下文、资源 ID、actor、trace 信息，也更适合记录审计事件。

### 5.3 保留什么样的“薄中心能力”是可以接受的

不保留在线 authz proxy，不代表所有中心能力都必须删除。可接受的薄能力包括：

- OpenFGA model / store 的管理工具；
- 后台 tuple 管理接口；
- 审计查询服务；
- 与 Keycloak、OpenFGA 相关的运维与管理控制台。

也就是说，**允许保留管理中心，不保留在线授权代理中心**。

---

## 6. 审计架构设计

### 6.1 总体原则

审计体系采用：

```text
业务服务本地产生审计事件 -> Kafka -> Audit Service -> ClickHouse / 查询 API
```

即：

- 审计事件的**产生是分布式的**；
- 审计事件的**消费、存储、查询是中心化的**。

### 6.2 为什么审计不要求中央 authz 服务

审计需要的是“统一汇聚”，不是“统一在线代理”。

对于授权场景，最有价值的审计数据往往只存在于业务服务本地，例如：

- operation 是什么；
- relation 是什么；
- object_type / object_id 是什么；
- actor 是谁；
- 为什么 allow / deny；
- 当前 request_id / trace_id 是什么。

这些信息让 **`pkg/authz` 成为最自然的审计锚点**。

### 6.3 审计事件的核心来源

第一阶段优先覆盖四类事件：

1. `authn.result`
   认证成功/失败、token 验证结果、principal 构建结果。

2. `authz.decision`
   OpenFGA check 的 allow / deny / no_rule / check_error 等决策结果。

3. `authz.tuple.changed`
   tuple 写入、删除、批量变更等授权关系变更。

4. `resource.mutation`
   关键业务资源的 create / update / delete 等变更行为。

### 6.4 审计锚点建议

#### P0：`pkg/authz.Authz`

这是最重要的统一锚点，应记录：

- actor；
- relation；
- object_type；
- object_id；
- operation；
- decision；
- error_code / reason；
- request_id / trace_id；
- cache_hit（若有）；
- service 名称。

#### P1：tuple 写删层

无论最终挂在：

- `pkg/openfga`，还是
- 某个上层 repo

都应统一记录 tuple 变更事件：

- write / delete；
- tuples 列表；
- operator principal；
- source service；
- request_id / trace_id。

#### P2：认证边界层

可在以下位置记录认证边界事件：

- 网关；
- `pkg/authn` / identity adapter；
- 特殊的 token verify 边界。

其目标不是替代业务审计，而是补足安全链路。

### 6.5 审计模型是否可扩展

结论：**必须可扩展，但不能失控。**

推荐模式：

- 固定骨架；
- typed detail；
- version；
- 少量 labels。

推荐的稳定骨架字段包括：

- `event_id`
- `event_type`
- `event_version`
- `occurred_at`
- `service`
- `operation`
- `actor`
- `target`
- `result`
- `error`
- `trace_id`
- `request_id`

detail 建议按类型扩展，例如：

- `AuthnDetail`
- `AuthzDetail`
- `TupleMutationDetail`
- `ResourceMutationDetail`

不建议一开始就把可扩展性完全做成任意 JSON / map。

---

## 7. actor 模型设计原则

### 7.1 actor 不直接等于 Keycloak claims

接入 Keycloak 后，Servora 内部仍不应让 actor 直接镜像 Keycloak token 结构。

原因：

1. Keycloak 是身份提供方，不是 Servora 的内部 canonical principal model；
2. 未来 actor 还可能来自 service account、内部 job、system actor、gateway header、甚至其他 IdP；
3. 如果 actor 直接与 Keycloak 字段绑定，内部模型会被外部协议反向约束。

### 7.2 actor 应作为通用 principal 模型

建议 actor 至少具备：

- `Type`：`user | service | anonymous | system`
- `ID`：内部稳定 principal id
- `Subject`：外部 subject（如 Keycloak `sub`）
- `ClientID`
- `Realm`
- `DisplayName`
- `Email`
- `Roles`
- `Scopes`
- `Attrs`

### 7.3 Keycloak 与 actor 的关系

应采用：

```text
Keycloak claims / gateway headers -> adapter -> actor.Actor
```

即：

- Keycloak claims 是输入协议；
- actor 是内部标准形态；
- gateway header 也是另一个输入源；
- `pkg/authz`、`pkg/audit`、业务服务只依赖 actor，不依赖 Keycloak 原始 claims 结构。

### 7.4 Keycloak 接口形态

Keycloak 的主集成方式是：

- OIDC / OAuth2 HTTP 端点；
- Admin REST API；
- SPI 扩展能力。

它不是一个 gRPC-first 的系统，因此 Servora 的适配设计应围绕：

- OIDC discovery
- JWKS
- token / introspection / userinfo
- Admin REST

来考虑，而不是围绕 gRPC 设计核心集成方式。

---

## 8. Servora 的框架化演进方向

### 8.1 总体方向

Servora 的目标不是单点实现 audit，而是构建一套可复用的微服务框架能力：

- `pkg/authz`
- `pkg/authn`
- `pkg/actor`
- `pkg/audit`
- `pkg/broker`
- `pkg/transport`
- 相关 proto / annotation / codegen

### 8.2 关于 `kratos-transport`

`/Users/horonlee/projects/go/kratos-transport` 对 Servora 有较高参考价值，但不应成为核心边界定义者。

推荐态度：

- **借鉴设计与实现思路**；
- **不直接作为 Servora 核心依赖接入**；
- 在 Servora 内部形成自己的 broker / transport / pkg 生态。

适合借鉴的内容包括：

- broker 抽象风格；
- producer / consumer option 组织方式；
- tracing / message / header / context 透传方式；
- 生命周期管理思路。

不建议直接继承的部分包括：

- 整套外部抽象边界；
- 将 Servora 自己的领域事件模型直接绑定在其 message 结构上；
- 为了支持 Kafka 先引入整套与当前需求不匹配的大抽象。

### 8.3 目录边界建议

建议长期拆分为：

#### `pkg/transport`

放请求/响应型或协议接入型能力：

- HTTP / gRPC / WebSocket / SSE / TCP 等；
- server/client middleware；
- metadata / header / context 透传；
- 服务接入协议封装。

#### `pkg/broker`

放消息型、事件型能力：

- broker interface；
- message / headers / publication；
- subscriber / producer / consumer lifecycle；
- topic / retry / ordering / tracing 等。

#### `pkg/task` 或 `pkg/queue`

若未来支持 Asynq 等任务队列，建议不要强塞进 `pkg/broker`。
原因是 job queue 与 event broker 在语义上不同：

- broker 偏事件总线；
- task / queue 偏延迟任务、重试任务、调度任务。

### 8.4 消息队列抽象策略

虽然 Kafka 是第一阶段实现，但框架层应预留多实现空间。

推荐节奏：

1. 先定义稳定的最小 broker 抽象；
2. 第一实现做 `pkg/broker/kafka`；
3. 后续再按语义补 NATS / RabbitMQ / Redis Streams 等；
4. 任务队列单独设计，不强并入 broker。

---

## 9. 审计的 all-in-proto 路线

### 9.1 总体判断

审计应像 authz 一样走：

```text
proto 注解 -> 代码生成 -> middleware 自动执行
```

这条路线非常适合 Servora，因为它符合当前框架已经具备的模式：

- 通用 proto 注解；
- 统一代码生成；
- 通用 middleware；
- 由 `make api` 驱动全链路更新。

### 9.2 推荐结构

#### 公共模型

放在根目录 `api/` 下，例如：

- `api/protos/audit/v1/audit.proto`
- `api/protos/audit/v1/annotations.proto`

其中：

- `audit.proto` 定义公共 `AuditEvent`、actor/target/detail 等模型；
- `annotations.proto` 定义 service / RPC 的 audit 注解规则。

#### 生成器

新增例如：

- `cmd/protoc-gen-servora-audit`

生成内容可包括：

- `operation -> audit rule map`
- 字段提取 helper
- detail builder helper
- middleware 可直接消费的 runtime rule 结构

#### 运行时

新增：

- `pkg/audit`

其职责包括：

- event builder；
- middleware；
- recorder / emitter；
- 与 `pkg/broker` 对接的 event publish runtime。

### 9.3 价值

这条路线的核心价值在于：

- 降低业务服务重复埋点成本；
- 将审计纳入 proto 驱动与代码生成体系；
- 让审计成为 Servora 的“框架能力”，而不是某个服务的附加实现。

---

## 10. 第五部分：Servora 现在该删什么、留什么、换什么

### 10.1 未来应逐步下线的部分

#### A. IAM 自己充当 issuer 的整套能力

包括：

- 自己暴露 JWKS；
- 自己暴露 OIDC discovery；
- 自己签发 / 刷新 access token；
- 自己实现登录、注册、密码重置、邮箱验证等完整认证闭环。

接入 Keycloak 后，这一层继续保留会造成“双认证中心”。

#### B. `Traefik -> IAM /v1/auth/verify -> 上游服务` 这条链

这条链在当前阶段是合理的过渡方案，但未来推荐改为：

- 网关直接对接 Keycloak 认证链路；
- 网关完成 token 验证；
- 网关把 principal 信息注入上游请求头；
- 业务服务不再依赖 IAM `/v1/auth/verify`。

#### C. 纯为早期 IAM 验证存在的本地身份域

由于当前 `user / org / project / application` 很大一部分不是正式业务模型，因此凡是主要服务于“自建认证中心”的测试性本地域模型，都不应成为未来架构包袱。

### 10.2 应保留并重构的部分

#### A. `pkg/authz`

必须保留，并升级为：

- 统一授权执行层；
- 统一授权审计采集点。

未来角色：

- 不负责认证；
- 只负责授权判定；
- 在授权决策处统一产出审计事件。

#### B. `pkg/openfga`

必须保留，作为所有业务服务访问 OpenFGA 的统一适配层。

未来统一承载：

- Check / CachedCheck；
- WriteTuples / DeleteTuples；
- cache / tracing / metrics / hooks；
- tuple 变更审计锚点。

#### C. `pkg/actor`

必须保留，并通用化为 canonical principal 模型。

#### D. `pkg/authn`

不要简单删除，但必须降级重构为**身份适配层**，而不是“认证中心能力”。

未来主要模式：

1. Gateway identity mode：从 header 构造 actor；
2. Direct JWT verification mode：只给极少数绕过网关的场景使用。

#### E. `pkg/transport/server/middleware/identity.go`

值得保留并增强，未来应支持更多 principal header，例如：

- `X-User-ID`
- `X-Client-ID`
- `X-Principal-Type`
- `X-Realm`
- `X-Roles`
- `X-Scopes`
- `X-Email`
- `X-Subject`

### 10.3 应新增的部分

#### A. `pkg/audit`

职责：

- 定义统一审计事件 runtime；
- event builder；
- recorder / emitter；
- middleware；
- 与 broker 对接。

#### B. `pkg/broker`

职责：

- 抽象发布订阅接口；
- message / headers / key / metadata；
- producer / consumer 运行时。

#### C. `pkg/broker/kafka`

作为第一实现，服务于 audit 与后续事件总线。

#### D. `app/audit/service`

作为中心化审计消费、存储、查询服务。

#### E. `cmd/protoc-gen-servora-audit`

作为审计注解的代码生成器，与 `protoc-gen-servora-authz` 并列。

---

## 11. 分阶段演进建议

### 第一阶段：定骨架

目标：不急着替换全部认证链路，先把框架骨架立起来。

建议：

1. 固化 Keycloak + Traefik + OpenFGA + Audit Service 的目标架构；
2. 设计 `actor` v2；
3. 设计 `audit.proto` / `annotations.proto`；
4. 设计 `pkg/broker` 最小抽象；
5. 新增 `pkg/audit` runtime 骨架。

### 第二阶段：先做审计主链

建议：

1. 落 `pkg/broker/kafka`；
2. 落 `pkg/audit`；
3. 在 `pkg/authz` 接入 `authz.decision` 事件；
4. 在 tuple 写删链路接入 `authz.tuple.changed` 事件；
5. 新建 `app/audit/service` 消费并落库。

### 第三阶段：完成 Keycloak 接入

建议：

1. 让网关改为对接 Keycloak；
2. 业务服务默认使用 identity header adapter；
3. 将 `pkg/authn` 改造成标准化身份适配层；
4. 清理 IAM 自建 issuer / verify 主链中的旧能力。

### 第四阶段：推进 all-in-proto

建议：

1. 新增 audit annotations；
2. 实现 `protoc-gen-servora-audit`；
3. 让 middleware 自动按 proto 规则生成审计事件；
4. 逐步减少手写 emit 逻辑。

### 第五阶段：扩展 Servora 生态

建议：

1. 补 `pkg/broker` 的更多实现；
2. 设计 `pkg/task` / `pkg/queue`；
3. 统一框架级 observability、eventbus、identity、audit、authz 能力；
4. 将 Servora 逐步沉淀为对外发布的微服务框架生态。

---

## 12. 最终结论

本次设计的核心不是“替换一个认证服务”，而是为 Servora 确立一套长期有效的边界：

- 认证交给 **Keycloak**；
- 网关负责 **统一认证与 principal 注入**；
- 授权由 **各业务服务本地执行 `pkg/authz` + OpenFGA**；
- 审计采用 **本地 emit + Kafka + 中心 Audit Service**；
- actor 设计为 **通用 principal 模型**；
- 审计与授权一样，逐步走向 **all-in-proto + 注解 + 代码生成 + middleware**；
- broker / transport / audit / authz / actor 将共同构成 **Servora 的 pkg 框架生态**。

这意味着，Servora 未来不再围绕“一个不断膨胀的中央 IAM 服务”演进，而是围绕：

- 明确的基础设施边界；
- 清晰的框架能力分层；
- 通用的 proto 驱动与代码生成能力；
- 面向微服务脚手架的长期 pkg 生态；

持续演进。
