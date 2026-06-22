# tmcopilot-project CLI Auth Onboarding Support Plan

## 1. 背景

`tmcopilot-cli` 已支持三种本地配置路径：

- `tmc setup`：普通入口。
- `tmc auth login --email <email> --password-stdin`：调用 `/auth/login`，再调用 `/auth/api-keys` 自动创建并保存 CLI API key。
- `tmc setup --api-key-stdin`：已有 API key 的脚本、CI、回退路径。

这个方案已经比手动配置环境变量简单，但还不是理想的普通用户体验。原因是 `/auth/login` 当前可能要求 `cf_turnstile_response`，而 CLI 不适合让用户手工复制 Turnstile token；同时密码从终端进入 CLI 也不如浏览器登录自然。

当前后端已经具备 API key 调 REST 的基础：REST middleware 支持 `Authorization: Bearer <api_key>` 和 `X-API-Key`，router 也已经注入 API key validator。因此 device-code/browser 登录不是 CLI 可用性的前置条件，而是普通用户 onboarding 体验升级。

术语说明：本文的 device-code/browser 登录不是 RFC 8628 OAuth Device Authorization Grant，不提供 OAuth `grant_type=device_code`、OAuth token endpoint 或 refresh token 语义。它只是借用“CLI 发起、浏览器授权、CLI 轮询”的用户体验，最终交付的是 TMCopilot 一次性可领取的 API key。

因此，`tmcopilot-project` 需要补一个浏览器/device-code 登录能力，让 CLI 的默认 onboarding 接近：

```bash
tmc setup
```

随后 CLI 打开浏览器或打印验证 URL，用户在网页完成登录、Turnstile、MFA、workspace 选择和授权，CLI 轮询拿到一次性 API key 并保存。

## 2. 非目标

- 不通过 MCP 完成 CLI 登录。
- 不复用 MCP OAuth handler 实现 CLI device flow；这是 REST CLI auth flow，接口固定落在 `/api/v1/auth/cli/*`。
- 不实现严格 OAuth Device Authorization Grant；不新增 OAuth device token endpoint、OAuth grant type 或 CLI refresh token。
- 不要求后端提供 CLI/Agent 命令元数据。
- 不引入独立于现有 REST `/api/v1` 的业务 API 体系。
- 不让 CLI 直接访问数据库或前端内部接口。
- 不改变大结果导出策略。
- 不要求保留旧接口稳定性分级；后端快速迭代时，CLI 跟随 Swagger 和契约测试更新。

## 3. 推荐用户流程

### 3.1 默认交互流程

```bash
tmc setup
```

CLI 行为：

1. 调用 `POST /api/v1/auth/cli/device-code` 创建登录会话。
2. 打开 `verification_uri_complete`。
3. 同时在终端显示 `user_code` 和 URL。
4. 轮询 `POST /api/v1/auth/cli/token`。
5. 授权完成后，CLI 的第一次合法轮询返回一次性 `raw_key`。
6. CLI 保存 API key 到本地 credentials。
7. CLI 调用 `/auth/me` 验证。

### 3.2 无浏览器或 Agent 流程

```bash
tmc auth login --no-browser
```

CLI 只打印 URL 和 user code，不主动打开浏览器，并持续轮询。

### 3.3 当前兼容流程

当前已实现的邮箱密码和 API key 方案保留为 fallback：

```bash
printf '%s' "$TMCOPILOT_PASSWORD" | tmc auth login --email user@example.com --password-stdin
printf '%s' "$TMCOPILOT_API_KEY" | tmc setup --api-key-stdin
```

## 4. 后端接口设计

### 4.1 创建 CLI 登录会话

```http
POST /api/v1/auth/cli/device-code
```

Auth：不需要登录。

Request：

```json
{
  "client_name": "tmcopilot-cli",
  "client_version": "0.1.0",
  "device_name": "MacBook-Pro",
  "requested_workspace_id": "",
  "key_name": "tmc cli MacBook-Pro",
  "api_key_expires_in": 0
}
```

Response：使用现有 REST envelope，示例中的字段位于 `data`。

```json
{
  "code": 0,
  "message": {
    "title": "OK",
    "text": "ok"
  },
  "data": {
    "device_code": "dc_550e8400-e29b-41d4-a716-446655440000_4f3c2d1a9b8e7f60",
    "user_code": "ABCD-EFGH",
    "verification_uri": "https://app.tmcopilot.example/cli/activate",
    "verification_uri_complete": "https://app.tmcopilot.example/cli/activate?user_code=ABCD-EFGH",
    "expires_in": 600,
    "interval": 5
  }
}
```

Response headers：

```http
Cache-Control: no-store
Pragma: no-cache
```

要求：

- `device_code` 只返回给 CLI。
- `device_code` 必须是不可猜测的长 token，建议使用 UUIDv4 加额外 64-128 bit 随机后缀，或直接使用 192-256 bit `crypto/rand` 生成的 base64url opaque token。
- `device_code` 不需要用户手工输入，长度优先满足安全性和碰撞风险，不为可读性妥协。
- `user_code` 用于网页确认。
- request 中的 `api_key_expires_in` 表示最终 API key TTL 请求值；response 中的 `expires_in` 表示 device session 剩余有效秒数，二者不能在 DTO 中混用。
- 后端只存 device code hash 和 user code hash，不存明文。
- 这里只创建 device session，不创建 API key。
- 创建响应包含可用于轮询的 `device_code`，必须和 raw key 成功响应一样禁缓存。
- 会话默认 10 分钟过期。
- 对 IP、device code、user code 做 rate limit。

### 4.2 浏览器授权页

页面建议：

```text
/cli/activate?user_code=ABCD-EFGH
```

行为：

1. 如果用户未登录，走现有网页登录流程。
2. 复用现有 Turnstile、MFA、账号状态校验。
3. 显示 CLI 请求信息：client、device、key name、workspace、过期时间。
4. 用户选择或确认 workspace。
5. 用户点击授权。
6. 后端校验用户对所选 workspace 有权限。
7. 后端只把 device session 标记为 `approved`，并记录 `user_id`、`workspace_id`、`key_name`、`requested_api_key_expires_in`、`effective_api_key_expires_in`、`approved_at`。
8. 浏览器授权阶段不创建 API key，因为当前 API key 的 `raw_key` 只在创建瞬间可得，库里只保存 hash。

`/cli/activate` 是前端页面路由，不直接承载授权状态机。页面必须调用下面三个受保护 REST API；这些 API 必须是 JWT-only web auth，不能走 public auth，不能接受 API key auth，也不能复用 CLI 轮询接口。

JWT-only 必须作为明确 router 任务落地：

1. 在 `backend/internal/protocol/rest/router.go` 中新增独立 `jwtOnlyV1` group，只挂 `middleware.AuthJWT`。
2. activation 三个接口只注册到 `jwtOnlyV1`，不要注册到现有 `AuthJWTOrAPIKey` protected group。
3. 不把 `auth_method` 分支作为首选方案；它只能作为未来兼容手段，首批实现以独立 JWT-only group 为准。

原因：现有普通 v1 protected group 同时接受 JWT 和 API key。如果 activation approve/deny 被 API key 放行，就会绕过“浏览器登录 + Turnstile/MFA”的设计目标，让已有 CLI key 授权新的 CLI 登录。

#### 4.2.1 查询待授权 session

```http
GET /api/v1/auth/cli/activation?user_code=ABCD-EFGH
```

Auth：必须是已登录网页登录态/JWT-only；API key 不可调用。

行为：

1. 归一化 `user_code`：trim、去掉 `-` 和空格、转大写。
2. 按 user code hash 查询 session。
3. 惰性处理过期 session。
4. 只返回浏览器授权页需要展示的信息，不返回 `device_code`、`raw_key` 或任何 API key 明文。

Response：使用现有 REST envelope，示例中的字段位于 `data`。

```json
{
  "code": 0,
  "message": {
    "title": "OK",
    "text": "ok"
  },
  "data": {
    "status": "pending",
    "user_code": "ABCD-EFGH",
    "client_name": "tmcopilot-cli",
    "client_version": "0.1.0",
    "device_name": "MacBook-Pro",
    "key_name": "tmc cli MacBook-Pro",
    "requested_workspace_id": "",
    "requested_api_key_expires_in": 0,
    "effective_api_key_expires_in": null,
    "expires_at": 1710000000,
    "workspace_options": [
      {
        "id": "workspace_123",
        "name": "Default Workspace"
      }
    ]
  }
}
```

如果 user code 无效、错误次数达到上限、已过期、已拒绝或已消费，仍返回 envelope，并在 `data.status` 中表达业务状态。未登录才返回现有 401 语义。

`workspace_options` MVP 可复用现有 `/auth/workspaces` 的可访问 workspace 列表行为，预期多数用户数量较小，可以全量返回并把默认 workspace 排在前面。如果未来 workspace 数量变大，再扩展分页或搜索参数；首批不为 activation 单独发明新的 workspace 查询语义。

#### 4.2.2 授权 session

```http
POST /api/v1/auth/cli/activation/approve
```

Auth：必须是已登录网页登录态/JWT-only；API key 不可调用。

Request：

```json
{
  "user_code": "ABCD-EFGH",
  "workspace_id": "workspace_123",
  "key_name": "tmc cli MacBook-Pro",
  "api_key_expires_in": 0
}
```

Response：

```json
{
  "code": 0,
  "message": {
    "title": "OK",
    "text": "ok"
  },
  "data": {
    "status": "approved",
    "expires_at": 1710000000,
    "effective_api_key_expires_in": 7776000,
    "workspace": {
      "id": "workspace_123",
      "name": "Default Workspace"
    }
  }
}
```

要求：

- 只允许 pending 且未过期 session 被 approve。
- 必须校验当前登录用户拥有 `workspace_id` 权限。
- 必须应用 CLI API key TTL 策略，把 `api_key_expires_in` 归一化为 `effective_api_key_expires_in`。
- 只标记 session 为 `approved`，并记录 `user_id`、`workspace_id`、`key_name`、`requested_api_key_expires_in`、`effective_api_key_expires_in`、`approved_at`。
- 不创建 API key，不返回 `raw_key`。

#### 4.2.3 拒绝 session

```http
POST /api/v1/auth/cli/activation/deny
```

Auth：必须是已登录网页登录态/JWT-only；API key 不可调用。

Request：

```json
{
  "user_code": "ABCD-EFGH",
  "reason": "user_cancelled"
}
```

Response：

```json
{
  "code": 0,
  "message": {
    "title": "OK",
    "text": "ok"
  },
  "data": {
    "status": "denied"
  }
}
```

要求：

- 只允许 pending 且未过期 session 被 deny。
- 记录 `denied_at` 和可审计的 `denied_reason`。
- 不创建 API key，不返回 `raw_key`。

### 4.3 CLI 轮询 token

```http
POST /api/v1/auth/cli/token
```

Auth：不需要登录。

Request：

```json
{
  "device_code": "dc_550e8400-e29b-41d4-a716-446655440000_4f3c2d1a9b8e7f60"
}
```

Pending response：使用现有 REST envelope，示例中的字段位于 `data`。

```json
{
  "code": 0,
  "message": {
    "title": "OK",
    "text": "ok"
  },
  "data": {
    "status": "authorization_pending",
    "interval": 5
  }
}
```

Success response：使用现有 REST envelope，示例中的字段位于 `data`。

```json
{
  "code": 0,
  "message": {
    "title": "OK",
    "text": "ok"
  },
  "data": {
    "status": "authorized",
    "raw_key": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
    "key": {
      "id": "key_123",
      "name": "tmc cli MacBook-Pro",
      "key_prefix": "01234567",
      "expires_at": 1717776000
    },
    "user": {
      "id": "user_123",
      "email": "user@example.com"
    },
    "workspace": {
      "id": "workspace_123",
      "name": "Default Workspace"
    }
  }
}
```

Polling statuses：

- `authorization_pending`
- `slow_down`
- `expired_token`
- `access_denied`
- `invalid_device_code`
- `already_consumed`

状态与 HTTP/pkg code 映射：

- device flow 的预期业务状态都返回 HTTP 200 + 现有 REST envelope，并通过 `data.status` 表达。
- `authorization_pending`、`slow_down`、`expired_token`、`access_denied`、`invalid_device_code`、`already_consumed` 都属于业务状态，不用 4xx 驱动 CLI 轮询分支。
- 只有 JSON 格式错误、字段类型错误、缺少必填字段这类 malformed request 返回 HTTP 400。
- 只有受保护的 activation API 未登录时返回 HTTP 401。
- workspace 无权限、账号锁定、账号禁用等浏览器授权失败场景可使用现有 403/error envelope；CLI token 端只观察最终业务状态。

要求：

- `raw_key` 只能被成功轮询返回一次。
- 返回 `raw_key` 的响应必须带 `Cache-Control: no-store`，兼容性需要时补 `Pragma: no-cache`。
- 第一次合法轮询 approved session 时，后端必须在同一个事务中创建 API key、记录 `api_key_id`、标记 session `consumed`、返回 `raw_key`。
- `raw_key` 示例必须跟随当前后端真实格式：32 字节随机值 hex 编码，长度 64，`key_prefix` 为前 8 位。引入 `tmc_` 前缀属于独立行为变更，不放入本计划默认范围。
- token success response 必须返回实际 API key 过期时间，例如 `key.expires_at`；CLI 以服务端返回值为准。
- API key 不做 workspace-scoped key；它仍然只绑定 user。CLI 保存 `workspace.id`，后续请求用现有 `X-Workspace-ID` header 传递 workspace。
- 后续轮询同一个 device code 返回 `already_consumed`，不能再次返回 raw key。
- polling interval 必须由服务端控制，CLI 遵守。

### 4.4 CLI API key 有效期策略

CLI device flow 不能让普通用户创建永久 API key。

建议策略：

- `api_key_expires_in` 缺省或为 `0` 表示使用服务端默认 TTL；如果为了兼容已有草案字段保留 `expires_in`，也必须只表示 API key TTL 请求值，不能和 device session TTL 混用。
- 服务端默认 TTL 建议 90 天。
- 服务端提供可配置最大 TTL，建议默认最大值 180 天。
- `api_key_expires_in` 大于最大 TTL 时返回 validation error，不静默放大或创建永久 key。
- 实际 API key 过期时间由 auth domain service 根据策略计算，不能信任 CLI 传入值。
- approve 时就要把请求值归一化为 `effective_api_key_expires_in` 并写入 session；如果 approval 后、consume 前服务端默认 TTL 配置变化，consume 仍使用 session 中已记录的 effective TTL。
- consume 创建 API key 后，把实际 `api_key_expires_at` 写回 session，并在 token success response 中返回实际过期时间。
- 如果产品确实需要永久 key，应保留在现有人工/API key 管理后台能力中，不能通过 CLI onboarding 默认创建。

### 4.5 撤销 CLI key

可以复用现有：

```http
DELETE /api/v1/auth/api-keys/{id}
```

如果需要更友好的 CLI logout，可补：

```http
POST /api/v1/auth/cli/revoke
```

Request：

```json
{
  "key_id": "key_123"
}
```

这不是首批必需接口；现有 API key revoke 足够支撑 CLI。

## 5. DDD 分层落点

### 5.1 Protocol Layer

建议新增：

```text
backend/internal/protocol/rest/handler/cli_auth.go
```

职责：

- HTTP 参数绑定。
- 调用 usecase。
- 返回统一 envelope。
- Swagger annotation。
- 在 router 中新增 `jwtOnlyV1` group，只挂 `middleware.AuthJWT`。
- activation 三个接口只注册到 `jwtOnlyV1`，确保 API key 无法调用。

禁止：

- 在 handler 中写 device session 业务逻辑。
- 在 handler 中直接创建 API key。
- 把 activation approve/deny 挂到普通 `AuthJWTOrAPIKey` protected group 后不区分 auth method。

### 5.2 Usecase Layer

建议新增：

```text
backend/internal/usecase/api/cli_auth.go
```

职责：

- 创建 device session。
- 校验 user code。
- 编排浏览器授权、CLI 轮询和 DTO 输出。
- 调用 auth domain service 完成状态迁移、interval 校验、消费和 API key 创建。
- 通过 workspace domain service 校验 workspace 权限，并在 token success response 中组装 workspace DTO。
- 输出 DTO。

Usecase 可以依赖：

- auth domain service。
- workspace 查询能力。
- clock/random/code generator abstractions。

禁止：

- usecase 直接操作 device session repository 改状态。
- usecase 直接实现 approved -> consumed 状态机。
- usecase 直接创建 CLI API key 并手动拼消费状态。

### 5.3 Domain Layer

建议新增或扩展：

```text
backend/internal/domain/auth/cli_device_session.go
backend/internal/domain/auth/repositories/cli_device_session_repository.go
backend/internal/domain/auth/services/cli_device_session.go
```

核心状态：

- `pending`
- `approved`
- `denied`
- `expired`
- `consumed`

核心不变量：

- 过期 session 不可授权。
- consumed session 不可再次返回 raw key。
- pending session 才能被 approve/deny。
- polling 太频繁返回 `slow_down`。
- approved -> consumed、`last_polled_at`、`slow_down` 判断和 API key 创建必须在同一个事务边界内完成。

建议 domain service 方法：

```go
CreateCLIDeviceSession(ctx, params) (deviceCode, userCode, session, error)
ApproveCLIDeviceSession(ctx, userCode, userID, workspaceID, keyName, requestedAPIKeyExpiresIn) (effectiveAPIKeyExpiresIn, error)
DenyCLIDeviceSession(ctx, userCode, userID, reason) error
PollCLIDeviceSession(ctx, deviceCode, now) (status, interval, error)
ConsumeApprovedCLIDeviceSession(ctx, deviceCode, now) (rawKey, key, session, error)
```

`ConsumeApprovedCLIDeviceSession` 是关键原子方法：

1. 用 `device_code_hash` 查 session 并加行锁，或用条件更新保证只有一个调用方能消费。
2. 校验未过期、状态为 `approved`、未 consumed。
3. 校验 polling interval，并更新 `last_polled_at`；太频繁返回 `slow_down`。
4. 在同一事务中调用现有 API key 创建能力，拿到一次性 `raw_key`。
5. 使用 session 中的 `effective_api_key_expires_in` 创建 API key。
6. 写入 `api_key_id`、`api_key_expires_at`、`consumed_at`、`status=consumed`。
7. 提交事务后向 usecase 返回 `raw_key`、API key 和 session；usecase 再按 `session.workspace_id` 查询 workspace DTO 并组装响应。

底层 repository 需要支持行锁或条件更新，例如 `SELECT ... FOR UPDATE`，或 `UPDATE ... WHERE status='approved' AND consumed_at IS NULL RETURNING ...`，以防并发轮询重复创建 API key。

事务接口需要在 domain/repository 层有明确落点，不能让 usecase 直接 `pool.Begin`、直接写 SQL，或先调用普通 `CreateAPIKey` 再手动更新 session。可选实现：

```go
type AuthTxRunner interface {
    WithinTx(ctx context.Context, fn func(ctx context.Context, repos AuthRepositories) error) error
}

type AuthRepositories interface {
    CLIDeviceSessions() CLIDeviceSessionRepository
    APIKeys() APIKeyRepository
}
```

如果当前 repository 基于 sqlc，tx runner 内部应使用生成代码的 `Queries.WithTx(tx)` 创建 tx-scoped queries，再组装 tx-scoped repositories 传给 domain service。或者由 `ConsumeApprovedCLIDeviceSession` 内部持有 tx runner。现有 API key 创建逻辑可以抽出 tx-aware 内部 helper，让“插入 API key hash、拿到一次性 raw key、更新 device session consumed 状态”处于同一事务。usecase 只调用一个 domain service 方法，不暴露事务细节。

## 6. 数据模型

建议表：

```sql
CREATE TABLE auth_cli_device_sessions (
  id BIGSERIAL PRIMARY KEY,
  device_code_hash TEXT NOT NULL UNIQUE,
  user_code_hash TEXT NOT NULL UNIQUE,
  status TEXT NOT NULL CHECK (status IN ('pending', 'approved', 'denied', 'expired', 'consumed')),
  user_id BIGINT REFERENCES auth_users(id) ON DELETE SET NULL,
  workspace_id BIGINT,
  api_key_id BIGINT REFERENCES auth_api_keys(id) ON DELETE SET NULL,
  client_name TEXT,
  client_version TEXT,
  device_name TEXT,
  key_name TEXT,
  requested_api_key_expires_in BIGINT,
  effective_api_key_expires_in BIGINT,
  api_key_expires_at BIGINT,
  interval_seconds INT NOT NULL DEFAULT 5,
  last_polled_at BIGINT,
  user_code_attempt_count INT NOT NULL DEFAULT 0,
  last_user_code_attempt_at BIGINT,
  approved_at BIGINT,
  denied_at BIGINT,
  denied_reason TEXT,
  consumed_at BIGINT,
  expires_at BIGINT NOT NULL,
  created_at BIGINT NOT NULL,
  updated_at BIGINT NOT NULL
);
```

索引：

- `device_code_hash` unique。
- `user_code_hash` unique。
- `status, expires_at` 用于清理。
- `user_id, created_at` 用于审计。
- `workspace_id` 只记录授权时选择的默认 workspace，API key 本身不绑定 workspace。

约束说明：

- `status` 必须有数据库 CHECK 约束，不能只靠应用层常量。
- `user_id`、`api_key_id` 是 nullable FK；pending 阶段为空，approve/consume 后逐步填充。
- `workspace_id` 只保存 BIGINT，不在 auth domain migration 中 FK 到 `workspaces`，避免 auth migration 物理依赖 workspace domain 表结构。
- workspace 访问权、当前可用性、软删除/禁用语义必须由 usecase 通过 workspace domain service 校验，不交给 auth repository/migration 处理。
- 如果未来确实要为 `workspace_id` 增加 FK，必须作为明确的数据库完整性例外写入 ADR，并规定 migration 顺序：`workspaces` 先于 auth CLI device session 表创建/迁移。
- `api_key_id` 指向 `auth_api_keys(id)`，key 被 revoke 时不删除 session 记录；如果未来物理删除 API key，`ON DELETE SET NULL` 保留 session 审计线索。

## 7. Workspace 语义

MVP 不做 workspace-scoped API key。

现有后端语义是：

- API key 鉴权只解析 user。
- workspace 通过现有 `X-Workspace-ID` 请求头进入 request context。

因此 CLI device flow 的 workspace 规则是：

1. 创建 device session 时可以带 `requested_workspace_id`，也可以为空。
2. 浏览器授权时必须展示并确认 workspace；如果为空，让用户选择一个可访问 workspace。
3. 后端必须校验授权用户拥有该 workspace 权限。
4. 成功轮询返回 `workspace.id` 和 `workspace.name`。
5. CLI 把 `workspace.id` 保存到当前 profile 的 `workspace_id`。
6. CLI 后续请求继续使用现有 `X-Workspace-ID` header，不需要新的 API key scope 机制。

## 8. 过期与清理

首批必须做惰性过期判断：

- 创建、授权、轮询、消费时都检查 `expires_at`。
- 如果 session 已过期，domain service 将状态迁移为 `expired`，返回 `expired_token`。
- 过期 session 不可 approve、deny 或 consume。

后续可以加定时清理任务：

- 定期删除或归档超过保留期的 `expired`、`denied`、`consumed` session。
- 保留期建议 7-30 天，取决于审计要求。

## 9. 安全要求

- device code、user code 都只存 hash。
- device code 使用长随机 token，至少 128 bit 熵；推荐 192-256 bit 熵。UUIDv4 可以作为组成部分，但不要使用 UUIDv7/递增 ID 这类带可预测结构的值作为唯一秘密。
- raw API key 不落库，只在成功轮询时返回一次。
- 任何包含 `device_code` 或 `raw_key` 的响应都必须设置 `Cache-Control: no-store`，兼容性需要时补 `Pragma: no-cache`。
- access log、error log 和 panic/recovery log 不得记录 response body 中的 `raw_key`、`device_code` 或完整 API key。
- 浏览器 activation API 永远不返回或展示 `raw_key`；只有 CLI token 成功响应能拿到一次性 `raw_key`。
- user code 使用固定 alphabet：`ABCDEFGHJKLMNPQRSTUVWXYZ23456789`，排除易混淆字符 `0/O/1/I`。
- user code 建议 8-10 位，展示为 `ABCD-EFGH` 或 `ABCDE-FGHIJ`。
- user code 比较前统一 trim、去掉 `-` 和空格、转大写。
- 生成 user code 时按 hash unique constraint 碰撞重试，超过固定次数返回 server error。
- 同一个 session 的 user code 错误输入次数需要限制，建议最多 5 次；超过后标记 `denied` 或要求重新发起 device session。
- 完全不存在的 user code 没有 session 可计数，不能创建伪 session；只按 IP、normalized user code hash、user agent 等维度做限流和审计。
- 轮询接口按 device code 和 IP 限流。
- 创建会话接口按 IP 限流。
- 授权页必须要求已登录用户确认。
- 如果账号被锁定、禁用、未激活，不能授权 CLI。
- 首批审计使用现有结构化 app log，不新增 audit table；如已有统一 audit service，可在同一事件点接入。
- 审计事件覆盖 create、approve、deny、consume、expire。
- 审计字段至少包含 event、session id、user id、workspace id、api_key_id、status、client_name、client_version、device_name、ip、user agent、created_at/approved_at/consumed_at 等时间字段。
- 审计日志不得记录 `device_code` 明文、`user_code` 明文、`raw_key` 或完整 API key。
- 返回给 CLI 的错误不能泄漏用户是否存在。

## 10. Swagger 与测试

需要补 Swagger：

- `POST /auth/cli/device-code`
- `POST /auth/cli/token`
- `GET /auth/cli/activation`
- `POST /auth/cli/activation/approve`
- `POST /auth/cli/activation/deny`
- 可选 `POST /auth/cli/revoke`

### 10.1 Domain 状态机单测

至少覆盖：

- 创建 session 成功。
- expired session。
- denied session。
- slow_down。
- user code 大小写、空格和 `-` 归一化。
- user code 错误次数达到上限。
- `api_key_expires_in=0` 使用默认 TTL。
- approve 后、consume 前默认 TTL 配置变化时，consume 使用已存储的 `effective_api_key_expires_in`。
- 超过最大 TTL 被拒绝。

### 10.2 Repository 并发/事务集成测试

至少覆盖：

- 授权后第一次合法轮询在同一事务中创建 API key、consume session，并只返回一次 raw key。
- 并发轮询同一个 approved session，只能有一个请求成功创建 API key。
- API key 创建失败时 session 不得被标记 consumed。
- session consumed 更新失败时 API key 创建必须回滚。
- `last_polled_at`、`slow_down` 和 consumed 状态更新具备并发一致性。
- consume 后写入的 `api_key_expires_at` 与实际创建的 API key 过期时间一致。

### 10.3 REST contract 测试

至少覆盖：

- `POST /auth/cli/device-code` response envelope 和字段。
- `POST /auth/cli/device-code` response 包含 `Cache-Control: no-store`。
- `GET /auth/cli/activation` 必须要求网页登录态/JWT-only，API key 不可调用。
- `POST /auth/cli/activation/approve` 必须要求网页登录态/JWT-only，API key 不可调用。
- `POST /auth/cli/activation/deny` 必须要求网页登录态/JWT-only，API key 不可调用。
- activation 三个接口注册在 `jwtOnlyV1` group，不注册到普通 `AuthJWTOrAPIKey` protected group。
- pending 轮询。
- invalid device code。
- 完全不存在的 user code 不创建 session，只触发限流/审计路径。
- workspace 权限校验失败。
- REST response 使用 `pkg.Response{data: ...}` envelope。
- 轮询业务状态使用 HTTP 200 + `data.status`。
- malformed JSON/字段错误使用 HTTP 400。
- 成功返回 `raw_key` 的响应包含 `Cache-Control: no-store`。
- 成功返回 `raw_key` 的示例和 schema 不要求 `tmc_` 前缀；当前 raw key 为 64 位 hex 字符串。
- token success response 返回实际 API key 过期时间。
- Turnstile/MFA 通过网页登录流程处理，不要求 CLI 提交 Turnstile token。

### 10.4 CLI smoke test

后端完成后至少覆盖：

- `tmc setup` 能打开或打印 activation URL。
- `tmc auth login --no-browser` 能打印 user code 并轮询。
- 授权完成后 CLI 保存 API key 和 workspace id。
- CLI 不打印 raw API key。
- `tmc auth status --check` 成功。

## 11. CLI 侧后续改动

后端完成后，CLI 再调整：

1. `tmc setup` 默认走 device-code/browser login。
2. `tmc auth login` 默认走 device-code/browser login。
3. `tmc auth login --email --password-stdin` 保留为 fallback。
4. 新增 `--no-browser`。
5. 新增 `--poll-timeout`、`--poll-interval`，但默认遵守服务端 interval。
6. 成功后仍保存 API key，不保存 JWT refresh token。
7. 成功后保存服务端返回的默认 `workspace.id` 到当前 profile，后续请求继续发送现有 `X-Workspace-ID`。
8. 文档把 API key import 降级为 CI/高级用法。

## 12. 验收标准

- 新用户只需要运行 `tmc setup` 即可完成本地配置。
- 不需要复制 API key。
- 不需要在 CLI 中输入 Turnstile token。
- CLI 不打印 raw API key。
- raw API key 只通过轮询接口返回一次。
- 并发轮询不会重复创建 API key。
- CLI 保存授权 workspace，并通过现有 `X-Workspace-ID` header 使用它。
- `tmc auth status --check` 成功。
- Swagger 中能看到新接口 schema。
- 后端契约测试覆盖全部状态分支。
