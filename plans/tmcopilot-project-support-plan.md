# TMCopilot Project Support Plan For CLI

## 1. 目标

这份计划只汇总 `tmcopilot-project` 为支持 `tmcopilot-cli` 需要配合确认或补齐的工作。它不是要求后端重做接口体系，也不是要求为 CLI 建一套独立 Open API 平台。

核心原则：

- 优先复用当前已有的 `/api/v1` REST API。
- 优先复用当前已有的 Swagger 文档。
- 优先复用当前已有的统一分页模型。
- CLI 不通过 MCP 调用业务能力。
- CLI 不直接访问数据库。
- CLI 不要求后端维护 Agent 专用元数据。
- CLI 的 Help、命令说明、参数映射和使用建议由 CLI 自己维护或从 Swagger 生成。
- 列表型大结果由 CLI 逐页读取并流式写本地文件，不要求后端一次性返回全部数据。

`tmcopilot-project` 的重点配合项是：

1. 确认现有 Swagger 对 CLI 首批命令是否足够完整。
2. 确认现有分页接口可被 CLI 逐页读取。
3. 补齐 CLI 必需但当前缺失的少量认证、当前用户、workspace、文件下载或长任务查询接口。
4. 在必要时改善错误响应，方便 CLI 转成结构化错误。
5. 增加针对 CLI 依赖接口的轻量契约测试，避免无意破坏。

## 2. 不做的事

以下内容不作为 `tmcopilot-project` 支持 CLI 的要求：

- 不做接口稳定性分级。项目处于快速迭代阶段，如果出现更好的接口，CLI 随后端和 Swagger 调整即可。
- 不做独立的 CLI endpoint registry。
- 不要求后端提供 CLI/Agent metadata，例如 recommended command、MCP fallback、agent hint。
- 不把 MCP guard 纳入 CLI 配套计划。MCP 是否限制大结果是 MCP 自身产品/技术问题，不是 CLI 的依赖项。
- 不要求后端提供 `fetch_all=true` 这类一次性返回所有列表数据的接口。
- 不要求后端为普通列表查询建设通用 Export/Artifact 系统。
- 不让 CLI 依赖前端页面行为或网页 Agent 行为。

## 3. 目标架构

CLI 的主调用链：

```text
tmcopilot-cli
      |
      v
TMCopilot REST API /api/v1
      |
      v
REST handlers
      |
      v
usecase/api
      |
      v
domain services / repositories / external adapters
```

CLI 的 schema/help 来源：

```text
Swagger/OpenAPI
      |
      v
CLI code generation or embedded command definitions
      |
      v
tmc help / command validation / examples
```

普通列表导出的数据流：

```text
CLI request page 1 -> write rows
CLI request page 2 -> append rows
CLI request page 3 -> append rows
...
CLI stops when page >= total_pages, max pages, max rows, or user cancels
```

关键点：CLI 可以提供类似 `--all` 或 `--page-all` 的用户体验，但其内部语义必须是逐页请求和流式写文件，不能在后端一次性拉全量，也不能在 CLI 内存里累计完整大数组后再写。

## 4. 当前可复用基础

当前 `tmcopilot-project` 已有这些基础，CLI 应直接复用：

- Swagger 文档：`backend/docs/swagger/`。
- REST 统一响应 envelope：`backend/internal/pkg/response.go`。
- REST 统一分页模型：`backend/internal/pkg/pagination.go`。
- API key 创建、列表、撤销：`backend/internal/protocol/rest/handler/auth.go`。
- MCP API key 鉴权实现可作为 Open API API key 鉴权参考：`backend/internal/protocol/mcp/auth_apikey.go`。
- 现有 REST handlers 和 usecase 层。
- 本地开发脚本可启动 API、MCP、listener、scheduler、frontend。

这意味着 CLI 初期不应该要求后端做大规模重构。更合理的做法是先基于 Swagger 和现有分页实现 CLI，然后只补实际缺口。

## 5. Swagger 配合项

### 5.1 目标

Swagger 是 CLI 识别 API 参数、响应结构和类型的主要来源。`tmcopilot-project` 需要保证 CLI 首批使用的接口在 Swagger 中可发现、可理解、可生成或可人工映射。

### 5.2 需要确认的内容

对 CLI 首批命令对应的接口，确认 Swagger 中包含：

- path。
- method。
- query parameters。
- request body schema。
- response envelope。
- data schema。
- auth requirement。
- common error responses。
- pagination fields。
- enum 值，如果接口使用 enum。

### 5.3 不要求 Swagger 表达的内容

以下内容由 CLI 自己维护，不要求后端 Swagger 表达：

- CLI 命令名称。
- CLI alias。
- CLI examples。
- CLI help text。
- CLI 默认字段展示。
- CLI 表格列顺序。
- CLI 文件输出建议。
- CLI 是否提示用户改用逐页导出。
- 面向 Agent 的使用说明。

### 5.4 Swagger 缺口处理方式

如果某个现有接口已经可用，但 Swagger 描述不完整，优先补 Swagger 注释或生成文档，而不是新增接口。

只有当现有 REST 接口本身缺失能力时，才考虑新增后端接口。

## 6. 分页配合项

### 6.1 目标

CLI 使用现有分页接口逐页读取数据。后端不需要提供全量读取接口。

### 6.2 需要确认的分页契约

CLI 首批列表接口应确认：

- 支持 `page`。
- 支持 `page_size` 或现有等价参数。
- 响应包含 `items`。
- 响应包含 `total`。
- 响应包含当前页信息。
- 响应可以判断是否还有下一页。
- 排序在同一查询条件下尽量稳定。

如果某些接口使用不同分页字段名，CLI 可以适配，但需要在计划中记录。

### 6.3 禁止的一次性全量读取

后端不需要新增：

```text
fetch_all=true
limit=999999
page_size=999999
```

CLI 也不应依赖这类参数。

### 6.4 CLI 逐页读取策略

后端只需保证普通分页接口可靠。CLI 负责：

- 控制 page loop。
- 控制 page size。
- 控制最大页数。
- 控制最大行数。
- 边读边写 CSV/JSONL/NDJSON。
- 遇到错误时保留已写文件并报告 partial failure。

### 6.5 建议确认的首批分页资源

- portfolio trademarks。
- portfolio office actions。
- portfolio conflict actions。
- CBP recordations。
- competitor activities。
- gap analyses。
- report list。
- trademark search results，如果当前搜索接口支持分页。

## 7. 认证和 Workspace 配合项

### 7.1 API Key

CLI 优先使用 API key。需要确认：

- API key 能调用 CLI 所需 REST API。
- API key 能解析到 user。
- API key last_used_at 可正常更新。
- API key 权限与当前用户一致。
- API key 可通过 `Authorization: Bearer <key>` 或既有 header 调用。

如果当前 REST middleware 只支持 JWT，不支持 API key，则需要补一个 API key auth middleware，复用现有 auth service 的 `ValidateAPIKey`。

### 7.2 当前用户

CLI 需要一个稳定方式获取当前身份。

优先复用现有接口。如果没有，建议补：

```text
GET /api/v1/auth/me
```

或使用当前项目已有等价接口。

返回至少包含：

```json
{
  "id": "user_...",
  "email": "user@example.com",
  "name": "User Name"
}
```

### 7.3 Workspace

如果当前系统存在 workspace 概念，CLI 需要知道：

- 当前用户有哪些 workspace。
- 默认 workspace 是哪个。
- 请求中如何指定 workspace。

优先复用已有接口。如果没有，建议补：

```text
GET /api/v1/auth/workspaces
```

或更贴合现有命名的等价路径。

CLI 请求可以用 header 或 query 指定 workspace，但后端只需要支持一种清晰方式即可。

建议：

```text
X-TMCopilot-Workspace-ID: <workspace-id>
```

如果现有系统已经有 workspace resolution 规则，CLI 应遵循现有规则，不要求新建一套。

## 8. 错误响应配合项

### 8.1 目标

CLI 可以先基于现有 response envelope、HTTP status 和 message 做错误分类。后端不需要一次性重做错误系统。

如果要提升 CLI 体验，可逐步补充 typed error 字段。

### 8.2 现有 Envelope

当前 envelope 可以继续使用：

```json
{
  "code": 40000,
  "message": {
    "title": "Bad Request",
    "text": "invalid page_size"
  }
}
```

CLI 可映射：

- 400 -> validation error。
- 401 -> auth error。
- 403 -> permission error。
- 404 -> not found。
- 429 -> rate limit。
- 500+ -> server error。

### 8.3 可选增强

如果后端愿意增强错误响应，可增加可选字段：

```json
{
  "error": {
    "type": "validation_error",
    "param": "page_size",
    "retryable": false,
    "hint": "Use a smaller page_size."
  }
}
```

这不是 CLI 第一阶段硬依赖。

### 8.4 Trace ID

建议后端在响应 header 中返回 trace ID：

```text
X-Trace-ID: <trace-id>
```

如果已有 trace middleware，CLI 只需读取并在错误中展示。

## 9. 文件和长任务接口

### 9.1 普通列表不需要后端 Export

对于列表型数据，例如 portfolio trademarks、office actions、competitor activities，后端不需要为了 CLI 新增通用 export 接口。

CLI 应通过普通分页接口逐页读取，并流式写入本地文件：

- CSV。
- JSONL / NDJSON。
- JSON array streaming writer。

### 9.2 需要后端文件能力的场景

后端只需要在这些场景提供文件或长任务接口：

- 后端本来就负责生成的报告。
- PDF / DOCX / XLSX 等服务端生成文件。
- 需要后端异步处理的复杂任务。
- 第三方文件下载代理。
- 已存在于系统中的 artifact。

### 9.3 建议确认的接口

优先确认现有接口是否已支持：

- report generate。
- report status。
- report download。
- gap analysis run。
- gap analysis status。
- generated file download。

如果没有，再按实际功能补接口，不建设泛化 Export/Artifact 平台作为 CLI 前置条件。

### 9.4 长任务通用响应

如果已有任务模型，CLI 复用现有模型。如果需要新增，建议最小化：

```json
{
  "task_id": "task_...",
  "status": "queued",
  "created_at": 1790000000
}
```

状态查询：

```json
{
  "task_id": "task_...",
  "status": "completed",
  "result": {
    "download_url": "https://..."
  }
}
```

## 10. CLI 首批接口清单

下面不是要求新增接口，而是 CLI 首批会优先寻找和复用的能力。实际路径以现有 Swagger 为准。

### 10.1 基础

- health。
- version。
- current user。
- API key list/create/revoke。
- workspace list 或默认 workspace 查询。

### 10.2 Portfolio

- portfolio summary。
- portfolio monitored summary。
- portfolio status counts。
- portfolio trademarks list。
- portfolio office actions list。
- portfolio conflict actions list。
- CBP recordations list。

### 10.3 Search

- trademark search。
- trademark details。
- TTAB case search。
- case search。
- office action document search。
- USPTO event documents。
- brand owner search。
- brand owner trademarks。
- lawyer search。
- lawyer contact info。

### 10.4 Competitor

- competitors list。
- competitor detail。
- competitor activities list。
- competitor scan results。
- latest competitor report。

### 10.5 Gap / Reports

- gap analyses list。
- gap analysis detail。
- gap analysis run，如果已存在。
- gap analysis result/export，如果已存在。
- reports list/detail/generate/download，如果已存在。

## 11. 测试配合项

### 11.1 Swagger 可用性测试

增加或确认：

- Swagger 生成在 CI 中不失败。
- CLI 首批使用的 endpoints 出现在 Swagger 中。
- 关键参数出现在 Swagger 中。
- 分页响应 schema 在 Swagger 中可见。

### 11.2 API 契约测试

针对 CLI 依赖接口增加轻量测试即可，重点不是冻结接口，而是防止无意破坏当前 CLI 使用的行为。

测试内容：

- auth 成功。
- auth 失败。
- 分页第一页。
- 分页第二页。
- 空列表。
- 常见 validation error。
- report/gap 长任务状态，如果 CLI 支持。

### 11.3 CLI 集成测试支持

本地 `make dev` 或等价脚本应能支持 CLI 跑基本集成测试：

- API health。
- API key auth。
- current user。
- portfolio list。
- search query。
- 一到两个分页读取场景。

不要求 MCP 测试作为 CLI 配套测试。

## 12. 文档配合项

后端文档不需要新增一套庞大的 CLI contract 文档。建议补最小必要说明：

- Swagger 访问路径。
- API key 使用方式。
- workspace 指定方式。
- 统一分页字段。
- 常见错误 envelope。
- 本地环境如何给 CLI 调用。
- 文件/报告下载接口，如果已有。

建议位置：

```text
backend/docs/cli-integration.md
```

## 13. 阶段计划

### Phase A: Swagger 和现有接口盘点

周期：3-5 天。

任务：

- 根据 CLI 首批命令列出现有 Swagger endpoint。
- 标记可直接使用的接口。
- 标记 Swagger 描述缺失但接口可用的项。
- 标记确实缺接口的项。
- 标记分页字段不一致的项。

验收：

- CLI 可以基于盘点结果开始实现首批 read commands。
- 后端缺口被明确为小任务，而不是重构项目。

### Phase B: Auth / Workspace 缺口补齐

周期：3-5 天。

任务：

- 确认 API key 能否调用 REST API。
- 如不能，补 API key auth middleware。
- 确认 current user 接口。
- 确认 workspace 查询/指定方式。

验收：

- CLI 可通过 API key 调用 REST API。
- CLI 可显示当前用户。
- CLI 可确定默认 workspace 或指定 workspace。

### Phase C: 分页接口确认

周期：3-5 天。

任务：

- 确认 portfolio trademarks 分页。
- 确认 office actions 分页。
- 确认 conflict actions 分页。
- 确认 CBP recordations 分页。
- 确认 competitor activities 分页。
- 确认 search results 分页能力。

验收：

- CLI 可以逐页读取首批列表资源。
- 没有任何首批命令依赖一次性全量读取。

### Phase D: 错误和 Trace 增强

周期：可选，3-5 天。

任务：

- 确认现有错误 envelope。
- 确认 HTTP status 是否合理。
- 可选增加 typed error 字段。
- 确认 trace ID header。

验收：

- CLI 能把常见错误映射成明确错误类型。
- 用户报错时能提供 trace ID。

### Phase E: 文件/长任务接口确认

周期：按现有功能决定。

任务：

- 盘点 report/gap/generated file 相关接口。
- 缺失时补最小 status/download 接口。
- 不为普通列表新增通用 export 系统。

验收：

- CLI 可下载后端生成的文件。
- CLI 可查询长任务状态。

## 14. 优先级

P0：

- Swagger endpoint 盘点。
- REST API key auth 确认。
- current user / workspace 确认。
- 首批列表接口分页确认。
- CLI 依赖接口的 Swagger 描述补齐。

P1：

- 常见错误响应增强。
- trace ID header 确认。
- report/gap 长任务接口确认。
- 文件下载接口确认。
- 轻量 API 契约测试。

P2：

- typed error 字段。
- 更完整的 Swagger schema。
- 更完整的 backend docs。

不列入 CLI 配套优先级：

- MCP guard。
- API 稳定性分级。
- CLI/Agent metadata registry。
- 普通列表通用 Export/Artifact 平台。

## 15. Definition Of Done

`tmcopilot-project` 配套完成的标准：

- CLI 首批命令都能在 Swagger 中找到对应接口，或有明确小缺口任务。
- CLI 能用 API key 调用 REST API。
- CLI 能获取当前用户。
- CLI 能确定 workspace 语义。
- CLI 首批 list 命令都能逐页读取。
- CLI 不依赖后端一次性全量返回。
- CLI 不依赖 MCP。
- 常见错误能被 CLI 映射。
- 本地开发环境可以跑 CLI 基础集成测试。

## 16. 首批 Backend Tickets 建议

1. Inventory Swagger endpoints required by first CLI commands.
2. Verify REST API key authentication for CLI.
3. Add REST API key middleware only if current REST auth does not support API keys.
4. Verify or expose current user endpoint.
5. Verify or expose workspace list/default workspace behavior.
6. Verify pagination for portfolio trademarks.
7. Verify pagination for office actions.
8. Verify pagination for conflict actions.
9. Verify pagination for CBP recordations.
10. Verify pagination for competitor activities.
11. Verify pagination for trademark search results.
12. Fill missing Swagger annotations for CLI-used endpoints.
13. Verify common error envelope and HTTP status mapping.
14. Add trace ID response header if not already present.
15. Verify report/gap generated file download APIs.
16. Add lightweight API contract tests for CLI-used endpoints.

## 17. 最终建议

`tmcopilot-project` 不需要为了 CLI 额外建立一套“稳定 Open API 平台”，也不需要把 MCP、大结果 guard、Agent 元数据放进 CLI 配套范围。

更合理的方案是：

1. CLI 直接使用现有 REST API 和 Swagger。
2. 普通列表数据由 CLI 逐页读取并流式写本地文件。
3. 后端只补实际缺口：REST API key、当前用户、workspace、Swagger 注释、错误/trace、已有报告或长任务下载。
4. CLI 的 Help、examples、命令映射和 Agent 使用建议留在 CLI 仓库内维护。

这样既符合初创项目快速迭代的现实，也能让 CLI 尽快落地，而不会把后端拖进一轮不必要的平台化重构。
