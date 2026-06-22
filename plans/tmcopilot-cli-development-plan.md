# TMCopilot CLI Development Plan

## 1. Executive Summary

TMCopilot CLI is a first-class command-line client for TMCopilot Open API. It is designed for humans, scripts, CI jobs, and external AI agents that need stable, machine-readable access to TMCopilot capabilities without being constrained by MCP context windows.

The CLI must not duplicate backend business logic. It should call versioned TMCopilot Open API endpoints, handle authentication and local configuration, provide consistent pagination and export behavior, and produce strict output and error contracts that AI agents can parse reliably.

The CLI complements MCP:

- MCP remains the tool discovery and small-response action protocol for external agents.
- CLI becomes the large-result, file-export, scripting, batch, and CI execution surface.
- Both surfaces share the same backend use cases through Open API contracts.

This is not an MVP plan. It is a complete product and engineering plan for a durable CLI that can be distributed, versioned, tested, and maintained as part of the TMCopilot platform.

## 2. Goals And Non-Goals

### 2.1 Goals

- Provide a stable CLI for TMCopilot Open API.
- Support humans and AI agents equally.
- Avoid MCP context overflow by supporting pagination, streaming-style NDJSON output, and file exports.
- Offer consistent output formats across all commands.
- Offer machine-readable structured errors.
- Support multiple profiles, endpoints, workspaces, and authentication methods.
- Support large datasets through `--page-all`, `--output`, and backend artifact/export endpoints.
- Support raw Open API calls for fast coverage of new backend endpoints.
- Support MCP bridge commands for diagnostics and compatibility, without using MCP as the primary data path.
- Ship installable binaries for macOS, Linux, and Windows.
- Ship npm wrapper installation for agent-friendly onboarding.
- Ship Agent Skills that teach external agents when and how to use the CLI.
- Establish CI, release, E2E, and security controls from the beginning.

### 2.2 Non-Goals

- Do not connect directly to TMCopilot databases.
- Do not embed backend domain logic in the CLI.
- Do not make MCP the CLI's main transport.
- Do not treat CLI output as user-interface text only; every default output must be parseable.
- Do not require local TMCopilot backend source code to use the CLI.
- Do not expose raw secrets in stdout, stderr, logs, config files, crash reports, or debug output.
- Do not rely on interactive prompts for agent workflows.

## 3. Product Positioning

### 3.1 Primary Users

- Attorneys and operations users who want terminal access to TMCopilot data.
- Power users who export trademarks, office actions, competitor activity, reports, and evidence tables.
- External AI agents such as Claude Code, Cursor, Gemini CLI, OpenAI Codex, and other terminal-capable agents.
- CI jobs and scheduled scripts that need repeatable access to TMCopilot.
- Internal engineers and customer success teams debugging customer data flows.

### 3.2 Usage Modes

- Human interactive mode:
  - Uses readable `table` and `pretty` formats.
  - May use prompts for setup and confirmation.
  - Can show progress and hints on stderr.

- Agent mode:
  - Uses `json`, `ndjson`, and explicit `--output`.
  - Avoids prompts.
  - Uses structured errors and hints.
  - Avoids dumping huge payloads into chat.

- Script/CI mode:
  - Uses environment variables or imported API keys.
  - Disables non-deterministic notices.
  - Relies on exit codes and stable JSON.

## 4. High-Level Architecture

### 4.1 Runtime Architecture

```text
Human / Script / External Agent
        |
        v
tmc CLI
        |
        v
TMCopilot Open API
        |
        v
Backend REST handlers
        |
        v
Use cases / domain services / repositories
```

### 4.2 Optional MCP Diagnostic Path

```text
Human / External Agent
        |
        v
tmc mcp ...
        |
        v
TMCopilot MCP server
        |
        v
Agent-runner MCP bridge
```

This path is only for tool listing, schema inspection, compatibility debugging, and targeted tool calls. It is not the primary CLI data path.

### 4.3 Repository Layout

```text
tmcopilot-cli/
  cmd/
    root.go
    auth/
    config/
    search/
    portfolio/
    officeaction/
    competitor/
    gap/
    reports/
    mcp/
    api/
    schema/
    doctor/
    completion/
  internal/
    agenthint/
    artifact/
    client/
    cmdutil/
    config/
    credential/
    errors/
    export/
    httptrace/
    output/
    pagination/
    schema/
    telemetry/
    validate/
    version/
  extension/
    credential/
  scripts/
    install.js
    run.js
  skills/
    tmc-shared/
    tmc-search/
    tmc-portfolio/
    tmc-reports/
    tmc-mcp/
  tests/
    cli_e2e/
      dryrun/
      live/
  docs/
    commands.md
    output-contract.md
    error-contract.md
  plans/
    tmcopilot-cli-development-plan.md
```

## 5. Technology Choices

### 5.1 Recommended Stack

- Language: Go.
- Command framework: Cobra.
- Configuration format: YAML or JSON.
- Key storage:
  - macOS Keychain.
  - Windows Credential Manager.
  - Linux Secret Service when available.
  - Environment variable fallback.
- HTTP: Go standard `net/http` with custom transport and retry middleware.
- Output:
  - Native JSON encoder.
  - Table renderer.
  - CSV writer.
  - NDJSON writer.
  - Optional jq-like filtering using `gojq`.
- Distribution:
  - GitHub Releases with checksums.
  - npm wrapper package.
  - Homebrew tap after binary releases are stable.

### 5.2 Why Go

- Backend is Go, so engineering conventions stay aligned.
- Single static binary is agent-friendly.
- Fast startup and minimal runtime dependencies.
- Cross-platform release workflow is straightforward.
- Robust standard library support for HTTP, JSON, CSV, filesystem, and testing.

## 6. Open API Contract Requirements

The CLI depends on TMCopilot Open API as its main contract. The backend must provide or stabilize the following API properties.

### 6.1 Versioning

Required:

- `/api/v1/version`
- `/api/v1/openapi.json`
- `/api/v1/schema/commands`
- `/api/v1/schema/resources`

API responses should include:

```json
{
  "code": 0,
  "message": {
    "title": "Success",
    "text": "ok"
  },
  "data": {}
}
```

The CLI should unwrap this envelope for user-facing data output unless `--raw-envelope` is passed.

### 6.2 Pagination

Every list endpoint used by CLI must support:

- `page`
- `page_size`
- `total`
- `total_pages`
- `items`
- stable ordering
- deterministic filters

Canonical response shape:

```json
{
  "items": [],
  "total": 1234,
  "page": 1,
  "page_size": 100,
  "total_pages": 13
}
```

CLI behavior:

- `--page` requests one page.
- `--page-size` controls size within allowed range.
- `--page-all` fetches all pages.
- `--page-limit` caps number of pages fetched by `--page-all`.
- `--page-delay` adds delay between pages.
- `--max-items` stops after enough records are collected.

### 6.3 Export And Artifact Endpoints

Backend should provide durable export APIs for results too large to stream through CLI stdout.

Required endpoints:

- `POST /api/v1/exports`
- `GET /api/v1/exports/{id}`
- `GET /api/v1/exports/{id}/download`
- `DELETE /api/v1/exports/{id}`

Suggested request:

```json
{
  "resource": "portfolio.trademarks",
  "format": "csv",
  "filters": {},
  "fields": [],
  "sort": {},
  "delivery": "artifact"
}
```

Suggested response:

```json
{
  "export_id": "exp_...",
  "status": "completed",
  "format": "csv",
  "row_count": 4821,
  "byte_size": 1048576,
  "download_url": "https://...",
  "expires_at": 1790000000
}
```

### 6.4 Schema Discovery

The CLI should not hard-code every field forever. It should support schema introspection.

Required schema metadata:

- Command name.
- Resource name.
- Parameters.
- Required fields.
- Types.
- Enum values.
- Pagination support.
- Export support.
- Risk level.
- Authentication requirements.
- Permission scope or entitlement key.

Example:

```json
{
  "name": "portfolio.trademarks.list",
  "resource": "portfolio.trademarks",
  "method": "GET",
  "path": "/api/v1/portfolio/trademarks",
  "risk": "read",
  "pagination": true,
  "export_formats": ["json", "csv", "ndjson", "xlsx"],
  "params": [
    {
      "name": "keyword",
      "type": "string",
      "required": false
    }
  ]
}
```

### 6.5 Error Contract

Backend errors should map cleanly into CLI typed errors.

API error shape should include:

```json
{
  "code": 40000,
  "message": {
    "title": "Bad Request",
    "text": "invalid page_size"
  },
  "error": {
    "type": "validation_error",
    "subtype": "invalid_argument",
    "param": "page_size",
    "hint": "Use a value between 1 and 1000.",
    "retryable": false
  }
}
```

If backend does not initially support this full shape, the CLI must classify existing HTTP status and response messages into its own typed error envelope.

## 7. CLI Command System

### 7.1 Top-Level Commands

```text
tmc auth
tmc config
tmc doctor
tmc search
tmc portfolio
tmc office-actions
tmc competitor
tmc gap
tmc reports
tmc exports
tmc mcp
tmc api
tmc schema
tmc completion
tmc version
```

### 7.2 Global Flags

All commands should support:

```text
--profile string
--endpoint string
--workspace string
--format json|pretty|table|csv|ndjson|markdown
--output string
--output-dir string
--fields string
--jq string
--raw-envelope
--no-color
--quiet
--verbose
--debug
--trace-id string
--timeout duration
--retry int
--retry-delay duration
```

List commands additionally support:

```text
--page int
--page-size int
--page-all
--page-limit int
--page-delay duration
--max-items int
--sort string
--sort-dir asc|desc
```

Write commands additionally support:

```text
--dry-run
--confirm
--yes
--idempotency-key string
```

### 7.3 Auth Commands

```bash
tmc auth login
tmc auth login --endpoint https://app.tmcopilot.ai --api-key-env TMCOPILOT_API_KEY
tmc auth import-key --name prod --api-key tmc_...
tmc auth status
tmc auth whoami
tmc auth logout
tmc auth api-keys list
tmc auth api-keys create --name "agent key" --expires-in 90d
tmc auth api-keys revoke --id key_...
```

Requirements:

- API keys should be stored in OS-native keychain by default.
- `TMCOPILOT_API_KEY` can override stored credentials.
- `--api-key` should never be echoed.
- `auth status --format json` must be parseable.
- `auth login` must have non-interactive mode for agents.

### 7.4 Config Commands

```bash
tmc config init
tmc config show
tmc config set endpoint https://...
tmc config set default_format json
tmc config profile list
tmc config profile use prod
tmc config profile add prod --endpoint https://...
tmc config profile remove prod
```

Config should include:

```yaml
current_profile: prod
profiles:
  prod:
    endpoint: https://app.tmcopilot.ai
    default_format: json
    default_workspace: ""
    api_version: v1
```

Secrets must not be stored in this config file.

### 7.5 Doctor Commands

```bash
tmc doctor
tmc doctor --format json
tmc doctor network
tmc doctor auth
tmc doctor mcp
```

Checks:

- CLI version.
- Latest version notice.
- Endpoint reachability.
- TLS validity.
- Auth status.
- Workspace resolution.
- Open API version compatibility.
- Export/artifact availability.
- MCP server availability when configured.

### 7.6 Search Commands

```bash
tmc search trademarks --keyword APOLLO --region us --limit 50
tmc search trademarks --owner "Apple Inc." --status live --class 9 --page-all --output apple.csv
tmc search trademarks --serial-number 90000000
tmc search trademarks --registration-number 6000000
tmc search ttab-cases --case-number 91200000
tmc search cases --party-name Nike --format table
tmc search office-action-documents --mark APOLLO --issue-type 2d
tmc search brand-owners --keyword "Apple"
tmc search lawyers --keyword "Jane Smith"
tmc search lawyer-contact --name "Jane Smith"
```

Requirements:

- For user-facing serial display, prefer raw serial number code over prefixed internal IDs.
- Support `--region us` default for trademark queries.
- Support owner, status, class, filing date, registration number, and serial number filters.
- Support `--fields` for export trimming.
- Support `--evidence` or `--include-evidence` where the backend provides evidence details.

### 7.7 Portfolio Commands

```bash
tmc portfolio summary
tmc portfolio monitored-summary
tmc portfolio status-counts
tmc portfolio trademarks list --page-all --output portfolio.json
tmc portfolio trademarks get --id tm_...
tmc portfolio office-actions list --stat overdue --format table
tmc portfolio conflict-actions list --risk high --page-all --output conflicts.csv
tmc portfolio cbp-recordations list --status active
```

Future write commands:

```bash
tmc portfolio trademarks add --serial-number 90000000 --dry-run
tmc portfolio trademarks remove --id tm_... --dry-run
tmc portfolio conflict-actions update --id ca_... --status reviewed --dry-run
```

Requirements:

- Read commands should be fully available before write commands.
- Write commands must support `--dry-run`.
- Destructive commands require `--confirm` or `--yes`.
- Commands should map closely to existing backend use cases.

### 7.8 Office Action Commands

```bash
tmc office-actions list --stat overdue
tmc office-actions list --serial 90000000
tmc office-actions documents list --serial 90000000
tmc office-actions documents content --serial 90000000 --document-page-id ... --output oa.txt
```

Requirements:

- Large document content defaults to file output.
- Inline content should be capped unless `--no-truncate` is explicitly passed.
- PDF or XML downloads should use artifact/download APIs where possible.

### 7.9 Competitor Commands

```bash
tmc competitor list
tmc competitor get --id comp_...
tmc competitor activities list --competitor-name Nike --page-all --output activities.ndjson
tmc competitor scan-results get --competitor-id comp_...
tmc competitor report latest
tmc competitor report latest --output report.md
```

Future write commands:

```bash
tmc competitor create --name "NewCo" --market US --dry-run
tmc competitor update --id comp_... --importance high --dry-run
tmc competitor archive --id comp_... --dry-run
```

### 7.10 Gap Analysis Commands

```bash
tmc gap list
tmc gap get --id gap_...
tmc gap run --base-company "Acme" --benchmark-company "Nike" --dry-run
tmc gap export --id gap_... --format xlsx --output gap.xlsx
```

Requirements:

- Long-running jobs should return job IDs.
- CLI should support polling:

```bash
tmc gap run ... --wait
tmc gap status --id gap_...
```

### 7.11 Reports And Exports Commands

```bash
tmc reports list
tmc reports get --id rpt_...
tmc reports generate trademark-search --input @params.json --dry-run
tmc reports generate trademark-search --input @params.json --wait --output report.docx
tmc exports create --resource portfolio.trademarks --format csv --filters @filters.json
tmc exports status --id exp_...
tmc exports download --id exp_... --output data.csv
```

Requirements:

- All report generation should be idempotent when possible.
- `--wait` should poll status with clear timeout.
- `--output` must write the final artifact to local disk.
- Without `--output`, output should be metadata only, not binary content.

### 7.12 MCP Commands

```bash
tmc mcp server-info
tmc mcp tools list
tmc mcp tools schema search_trademarks
tmc mcp call search_trademarks --args @args.json
```

Requirements:

- MCP commands use configured MCP endpoint or derive it from profile.
- MCP commands are diagnostic and compatibility tools.
- `tmc mcp call` must warn when result is large and suggest Open API command equivalents.
- MCP tool results should support `--output`.

### 7.13 Raw API Commands

```bash
tmc api GET /api/v1/version
tmc api GET /api/v1/portfolio/trademarks --params '{"page":1,"page_size":20}'
tmc api POST /api/v1/exports --data @export.json
```

Requirements:

- Use same auth/profile/config as other commands.
- Support `--params`, `--data`, `--data @file`, and stdin.
- Preserve full response when `--raw-envelope` is passed.
- Provide typed errors.

### 7.14 Schema Commands

```bash
tmc schema
tmc schema portfolio.trademarks.list
tmc schema search.trademarks
tmc schema --format markdown --output commands.md
```

Requirements:

- Pull schema from backend when available.
- Fall back to embedded schema for offline help.
- Include supported filters, output fields, export formats, and risk levels.

## 8. Output Contract

### 8.1 stdout

stdout is data only.

Examples:

```json
{
  "ok": true,
  "data": {
    "items": [],
    "total": 0
  }
}
```

For `--format csv`, stdout is CSV only.

For `--format ndjson`, stdout is newline-delimited JSON records only.

### 8.2 stderr

stderr is for:

- progress
- warnings
- hints
- debug traces
- structured errors

No command should write non-data prose to stdout.

### 8.3 JSON Envelope

Default successful JSON output:

```json
{
  "ok": true,
  "data": {},
  "meta": {
    "profile": "prod",
    "endpoint": "https://app.tmcopilot.ai",
    "trace_id": "tr_...",
    "page": 1,
    "page_size": 100,
    "total": 1234,
    "total_pages": 13
  }
}
```

For list commands with `--page-all`, metadata should include:

```json
{
  "pages_fetched": 13,
  "returned_count": 1234,
  "truncated": false
}
```

### 8.4 Table Output

Table output is for humans. It should not be the default in agent mode.

Rules:

- Stable column order.
- No wrapping by default unless terminal is interactive.
- Large text fields are truncated.
- Must provide hint to use `--format json` or `--output`.

### 8.5 CSV Output

CSV rules:

- Include header row.
- Normalize date fields to ISO 8601 where possible.
- Preserve raw serial numbers and registration numbers.
- Avoid prefixed internal display IDs unless explicitly requested.
- Escape fields using standard CSV writer.

### 8.6 NDJSON Output

NDJSON rules:

- One item per line.
- Metadata may be printed as a final object only if `--include-meta` is passed.
- Default NDJSON should be records only for pipeline compatibility.

### 8.7 File Output

`--output` behavior:

- Parent directory must exist unless `--create-dirs` is passed.
- Existing files are not overwritten unless `--overwrite` is passed.
- On success, stdout should return metadata in JSON mode.
- File write progress goes to stderr.

Example:

```json
{
  "ok": true,
  "data": {
    "path": "portfolio.csv",
    "format": "csv",
    "row_count": 4821,
    "byte_size": 1048576
  }
}
```

## 9. Error Contract

### 9.1 Exit Codes

```text
0  success
1  general failure
2  validation error
3  auth error
4  permission error
5  internal error
6  network error
7  API/server error
8  rate limit
9  file error
10 user cancelled
11 partial failure
12 version incompatibility
```

### 9.2 Error Envelope

stderr should receive structured JSON when `--format json` or agent mode is active.

```json
{
  "ok": false,
  "type": "validation_error",
  "subtype": "invalid_argument",
  "param": "--page-size",
  "message": "page size must be between 1 and 1000",
  "hint": "retry with --page-size 1000 or use --page-all",
  "retryable": false,
  "trace_id": "tr_..."
}
```

### 9.3 Error Categories

- `validation_error`
- `auth_error`
- `permission_error`
- `rate_limit`
- `network_error`
- `api_error`
- `server_error`
- `file_error`
- `version_error`
- `internal_error`
- `cancelled`
- `partial_failure`

### 9.4 Agent Recovery Metadata

Errors should include machine-readable recovery fields when possible:

```json
{
  "missing_scope": "portfolio:read",
  "valid_values": ["json", "table", "csv", "ndjson"],
  "suggestions": ["--page-size", "--page-all"],
  "retry_after_seconds": 30
}
```

## 10. Security Plan

### 10.1 Secrets

- Store API keys in OS-native keychain.
- Never store raw API keys in config file.
- Never print credentials in stdout or stderr.
- Redact:
  - `Authorization`
  - `X-API-Key`
  - `api_key`
  - `access_token`
  - `refresh_token`
  - `secret`
  - `password`

### 10.2 Input Safety

- Validate paths before reading `@file`.
- Validate output paths before writing.
- Block writing outside allowed paths only if strict mode is configured.
- Protect against terminal escape injection by sanitizing untrusted text in table/pretty output.

### 10.3 Write Operation Safety

- All create/update/delete commands support `--dry-run`.
- Destructive commands require `--confirm` or `--yes`.
- Dry-run should display request method, URL, normalized parameters, and expected risk.
- Write operations should include idempotency keys when supported.

### 10.4 Agent Safety

- Agent Skills must instruct agents:
  - Do not reveal API keys.
  - Use `--output` for large datasets.
  - Prefer `--dry-run` before write commands.
  - Do not paste full exported data into chat.
  - Summarize file paths, row counts, and key findings.

## 11. Backend Work Required

### 11.1 Open API Stabilization

Backend team must classify current `/api/v1` endpoints into:

- Public Open API stable.
- Internal frontend-only API.
- Deprecated API.
- Missing API needed by CLI.

Deliverables:

- Open API route inventory.
- Stable endpoint naming conventions.
- Pagination consistency audit.
- Error shape audit.
- Swagger/OpenAPI generated spec verified in CI.

### 11.2 CLI-Specific Endpoint Gaps

Likely required additions:

- `/api/v1/openapi.json`
- `/api/v1/schema/commands`
- `/api/v1/exports`
- `/api/v1/exports/{id}`
- `/api/v1/exports/{id}/download`
- `/api/v1/me/workspaces`
- `/api/v1/mcp/server-info`
- `/api/v1/mcp/tools`
- `/api/v1/mcp/tools/{name}/schema`

### 11.3 MCP Improvements

Required MCP changes:

- Add result size guard.
- Add `return_mode` convention for large result tools:
  - `inline`
  - `summary`
  - `artifact`
- Do not allow `fetch_all` to dump unbounded text through MCP by default.
- Return digest plus CLI suggestion when result is too large.

Example MCP hint:

```json
{
  "summary": "Found 4821 portfolio trademarks. Result is too large for MCP inline response.",
  "data": {
    "result_mode": "digest",
    "total": 4821,
    "suggested_cli": "tmc portfolio trademarks list --page-all --output portfolio.csv"
  }
}
```

## 12. Agent Skills Plan

### 12.1 Skill Packages

Create these skills in `skills/`:

- `tmc-shared`
- `tmc-search`
- `tmc-portfolio`
- `tmc-reports`
- `tmc-mcp`

### 12.2 tmc-shared

Content:

- Install instructions.
- Auth setup.
- Profile usage.
- Output contract.
- Security rules.
- Large result handling.
- Error recovery rules.

### 12.3 tmc-search

Content:

- Trademark search patterns.
- US default jurisdiction.
- Serial/registration number handling.
- Owner/class/status filters.
- When to use CLI instead of MCP.
- Export examples.

### 12.4 tmc-portfolio

Content:

- Portfolio summary.
- Portfolio trademarks list.
- Office actions.
- Conflict actions.
- CBP recordations.
- Large-dataset analysis workflow.

### 12.5 tmc-reports

Content:

- Report generation.
- Export status polling.
- Artifact downloads.
- File naming conventions.
- Summary expectations.

### 12.6 tmc-mcp

Content:

- Tool discovery.
- MCP diagnostics.
- MCP-to-CLI fallback.
- Large-result warnings.

## 13. Testing Strategy

### 13.1 Unit Tests

Required unit tests:

- Config loading and merging.
- Profile resolution.
- Credential provider resolution.
- HTTP request building.
- API envelope parsing.
- Error classification.
- Pagination loops.
- Output formatters.
- CSV escaping.
- NDJSON output.
- File output safety.
- Flag validation.
- Command schema rendering.

### 13.2 Contract Tests

Contract tests should run against recorded Open API fixtures.

Test cases:

- Success envelope.
- Backend validation error.
- Auth error.
- Permission error.
- Rate limit.
- Pagination with multiple pages.
- Empty list.
- Large export metadata.
- Artifact download.

### 13.3 Dry-Run E2E Tests

Dry-run tests should validate:

- Commands build expected HTTP method and path.
- Query params are correct.
- Request body is correct.
- No live mutations are performed.

### 13.4 Live E2E Tests

Live tests should run against a controlled test environment.

Required live flows:

- `auth status`
- `doctor`
- trademark search
- portfolio trademarks list
- office actions list
- competitor activities list
- raw API GET
- export create/status/download
- report generate/status/download when backend is ready

### 13.5 Agent E2E Tests

Run scripted agent tasks:

- Search owner trademarks and export CSV.
- List all portfolio trademarks and summarize counts.
- Fetch competitor activities and write NDJSON.
- Diagnose auth failure and retry after key import.
- Handle large result without pasting it into final answer.

### 13.6 Cross-Platform Tests

CI matrix:

- macOS arm64
- macOS amd64
- Linux amd64
- Linux arm64
- Windows amd64

## 14. CI/CD And Release Plan

### 14.1 CI Checks

Every PR:

- `go test ./...`
- `go vet ./...`
- `gofmt -l .`
- `go mod tidy` check
- static lint
- unit test coverage
- CLI command snapshot tests
- generated docs check

### 14.2 Release Artifacts

Each release should publish:

- `tmc-<version>-darwin-arm64.tar.gz`
- `tmc-<version>-darwin-amd64.tar.gz`
- `tmc-<version>-linux-amd64.tar.gz`
- `tmc-<version>-linux-arm64.tar.gz`
- `tmc-<version>-windows-amd64.zip`
- `checksums.txt`
- `SBOM`
- npm package metadata

### 14.3 npm Wrapper

TMCopilot CLI should follow the Lark/Feishu CLI distribution route:

- npm/npx is the cross-platform bootstrapper.
- GitHub Releases hold native Go binaries.
- Windows uses a native `tmc.exe` inside a zip archive.
- macOS/Linux use tar.gz archives.
- WSL is supported only as a fallback for technical users, not as the default Windows path.

Package:

```json
{
  "name": "@tmcopilot/cli",
  "bin": {
    "tmc": "scripts/run.js"
  }
}
```

Commands:

```bash
npx @tmcopilot/cli@latest install
npx @tmcopilot/cli@latest update
tmc version
```

Wrapper behavior:

- Detect `process.platform` and `process.arch`.
- Map `win32` to `windows`, `x64` to `amd64`.
- Download `tmc-<version>-<os>-<arch>.tar.gz` or `tmc-<version>-windows-<arch>.zip` from GitHub Releases.
- Download `checksums.txt` from the same release and verify SHA-256 before installation.
- Store the native binary under package-local `bin/tmc` or `bin/tmc.exe`.
- `scripts/run.js` invokes the native binary and auto-installs it if missing.
- `npx @tmcopilot/cli@latest install` is the recommended install path for macOS, Linux, and Windows.
- PowerShell installer, winget, Scoop, and Chocolatey are later convenience channels, not the primary initial route.

### 14.4 Version Compatibility

The CLI should check backend compatibility:

- CLI minimum backend version.
- Backend Open API version.
- Feature flags for export/schema/MCP bridge.

If incompatible:

```json
{
  "ok": false,
  "type": "version_error",
  "message": "backend does not support exports API",
  "hint": "upgrade TMCopilot backend or use a command without --output"
}
```

## 15. Observability

### 15.1 Client-Side Logging

By default:

- No log files.
- Minimal stderr progress only when interactive.

With `--debug`:

- Print request method/path.
- Print response status.
- Print trace ID.
- Redact secrets.

With `--trace`:

- Include timing breakdown.
- Include retry attempts.

### 15.2 Backend Trace Integration

The CLI should send:

- `X-TMCopilot-CLI-Version`
- `X-TMCopilot-CLI-Command`
- `X-TMCopilot-Trace-ID`
- `User-Agent: tmcopilot-cli/<version>`

Backend should return:

- `X-Trace-ID`
- request ID where available

## 16. Implementation Phases

### Phase 0: Detailed Design And Backend Inventory

Duration: 1 week.

Deliverables:

- CLI command inventory.
- Backend endpoint inventory.
- Open API gap list.
- Output contract doc.
- Error contract doc.
- Security design doc.
- Release design doc.

Acceptance criteria:

- Every planned CLI command has a mapped API endpoint or explicit backend gap.
- Backend gaps are filed as implementation tasks.
- Command naming and global flags are frozen for Phase 1.

### Phase 1: CLI Foundation

Duration: 1.5 weeks.

Scope:

- Go module setup.
- Cobra root command.
- Global flags.
- Config system.
- Profile system.
- Output package.
- Error package.
- HTTP client package.
- Version command.
- Completion command.
- Unit test framework.

Commands:

```bash
tmc version
tmc config init
tmc config show
tmc config profile list
tmc completion zsh
```

Acceptance criteria:

- Commands run on macOS/Linux/Windows.
- JSON output envelope is stable.
- Structured errors are emitted.
- Unit tests cover config, output, errors, and HTTP request construction.

### Phase 2: Auth And Doctor

Duration: 1 week.

Scope:

- Credential provider chain.
- API key import.
- Environment variable override.
- Keychain storage.
- Auth status.
- API key management through Open API.
- Doctor checks.

Commands:

```bash
tmc auth import-key
tmc auth status
tmc auth whoami
tmc auth logout
tmc auth api-keys list
tmc auth api-keys create
tmc auth api-keys revoke
tmc doctor
```

Acceptance criteria:

- API key is not stored in plaintext config.
- `TMCOPILOT_API_KEY` works in CI.
- `doctor --format json` is parseable.
- Auth failures return typed errors.

### Phase 3: Read Commands - Search And Portfolio

Duration: 2 weeks.

Scope:

- Search trademarks.
- Search TTAB cases.
- Search cases.
- Search office action documents.
- Search brand owners.
- Search lawyers.
- Portfolio summary.
- Portfolio trademarks list.
- Portfolio office actions list.
- Portfolio conflict actions list.
- CBP recordations list.

Commands:

```bash
tmc search trademarks
tmc search ttab-cases
tmc search cases
tmc search office-action-documents
tmc search brand-owners
tmc search lawyers
tmc portfolio summary
tmc portfolio trademarks list
tmc portfolio office-actions list
tmc portfolio conflict-actions list
tmc portfolio cbp-recordations list
```

Acceptance criteria:

- Every list command supports consistent pagination.
- Every list command supports `json`, `table`, `csv`, and `ndjson` where applicable.
- `--page-all --output` can export thousands of rows without stdout overflow.
- Dry-run E2E verifies paths and params.

### Phase 4: Raw API And Schema

Duration: 1.5 weeks.

Scope:

- Raw API command.
- Schema command.
- OpenAPI loading.
- Command schema rendering.
- Unknown command/flag suggestions.

Commands:

```bash
tmc api GET /api/v1/version
tmc api POST /api/v1/exports --data @body.json
tmc schema
tmc schema portfolio.trademarks.list
```

Acceptance criteria:

- Raw API supports GET/POST/PUT/PATCH/DELETE.
- `@file` and stdin input work.
- Schema output is available as JSON and Markdown.
- Agent can recover from unknown flag using suggestions.

### Phase 5: Export And Artifact System

Duration: 2 weeks.

Scope:

- Backend export endpoint integration.
- Export create/status/download.
- Local output file handling.
- Artifact metadata rendering.
- Polling with `--wait`.
- Large file download with progress on stderr.

Commands:

```bash
tmc exports create
tmc exports status
tmc exports download
tmc portfolio trademarks export
tmc competitor activities export
```

Acceptance criteria:

- Exports can write CSV, JSON, NDJSON, and XLSX when backend supports them.
- Existing files are protected unless `--overwrite`.
- Export metadata includes row count and byte size.
- Network interruption returns retryable typed error.

### Phase 6: Competitor, Gap Analysis, Reports

Duration: 2 weeks.

Scope:

- Competitor profile and activity commands.
- Gap analysis list/get/run/export.
- Report list/get/generate/download.
- Long-running job polling.

Commands:

```bash
tmc competitor list
tmc competitor get
tmc competitor activities list
tmc competitor scan-results get
tmc competitor report latest
tmc gap list
tmc gap get
tmc gap run
tmc gap export
tmc reports list
tmc reports generate
tmc reports download
```

Acceptance criteria:

- Long-running operations support `--wait`, `--timeout`, and status checks.
- Report binary output never goes to stdout by default.
- Agent Skills document correct report/export usage.

### Phase 7: MCP Bridge And Skills

Duration: 1.5 weeks.

Scope:

- MCP server info.
- MCP tools list.
- MCP schema.
- MCP tool call.
- Skills package authoring.
- Skills install/update command or documentation.

Commands:

```bash
tmc mcp server-info
tmc mcp tools list
tmc mcp tools schema search_trademarks
tmc mcp call search_trademarks --args @args.json
```

Acceptance criteria:

- MCP commands authenticate successfully.
- Large MCP responses warn and suggest CLI/Open API equivalents.
- Skills pass manual tests in at least three agent environments.

### Phase 8: Write Commands And Dry-Run Framework

Duration: 2 weeks.

Scope:

- Generic dry-run renderer.
- Create/update/delete command framework.
- Initial write commands for low-risk workflows.
- Confirmation policy.
- Idempotency keys.

Candidate commands:

```bash
tmc competitor create --dry-run
tmc competitor update --dry-run
tmc portfolio conflict-actions update --dry-run
tmc reports generate --dry-run
```

Acceptance criteria:

- Every write command supports `--dry-run`.
- Destructive commands require confirmation.
- Dry-run output is machine-readable.
- Live E2E creates, verifies, and cleans up test data.

### Phase 9: Distribution, Hardening, And Public Readiness

Duration: 2 weeks.

Scope:

- GitHub Releases.
- npm wrapper.
- Checksums.
- SBOM.
- Install/update flow.
- Homebrew draft.
- Full docs.
- E2E matrix.
- Security review.

Acceptance criteria:

- `npx @tmcopilot/cli@latest install` works.
- Binary checksums are published.
- Release notes are generated.
- CLI can be installed and used from a clean machine.
- All planned E2E tests pass against test backend.

## 17. Milestone Timeline

Assuming one primary CLI engineer and one backend engineer:

```text
Week 1:     Phase 0
Week 2-3:   Phase 1
Week 4:     Phase 2
Week 5-6:   Phase 3
Week 7-8:   Phase 4 and Phase 5 backend start
Week 9-10:  Phase 5
Week 11-12: Phase 6
Week 13:    Phase 7
Week 14-15: Phase 8
Week 16-17: Phase 9
```

With two CLI engineers and one backend engineer, the plan can compress to roughly 10-12 weeks by parallelizing:

- CLI foundation/auth.
- Backend Open API/export work.
- Command implementation.
- Skills/docs/tests.

## 18. Team Responsibilities

### 18.1 CLI Engineer

- CLI framework.
- Commands.
- Output/error contracts.
- Credential storage.
- Release pipeline.
- Unit tests.
- CLI E2E tests.

### 18.2 Backend Engineer

- Open API stabilization.
- Schema endpoints.
- Export/artifact endpoints.
- Pagination consistency.
- Error shape improvements.
- MCP result-mode improvements.

### 18.3 Agent/Docs Engineer

- Agent Skills.
- Command docs.
- Usage examples.
- Agent E2E workflows.
- Troubleshooting docs.

### 18.4 QA / Internal Dogfood

- Cross-platform installation.
- Large result exports.
- Auth and profile workflows.
- CI/script use cases.
- Failure recovery cases.

## 19. Definition Of Done

The CLI is production-ready when:

- Installation works through GitHub Releases and npm wrapper.
- API key auth works without plaintext credential storage.
- All planned read commands are implemented.
- Export/artifact workflow is implemented.
- Output formats are stable and documented.
- Error contract is stable and documented.
- `--page-all --output` handles large datasets without context overflow.
- Raw API command exists.
- Schema command exists.
- MCP bridge diagnostics exist.
- Agent Skills are available.
- CI covers unit, contract, dry-run E2E, and live E2E.
- At least three external agent environments have been manually tested.
- Security review is complete.
- Docs include installation, auth, command reference, output contract, error contract, and agent usage.

## 20. Key Risks And Mitigations

### 20.1 Backend API Instability

Risk:

- Frontend-oriented endpoints may change and break CLI.

Mitigation:

- Establish Open API stable routes.
- Add contract tests.
- Version command schemas.

### 20.2 Large Result Memory Usage

Risk:

- `--page-all` could accumulate too much memory before writing output.

Mitigation:

- Stream NDJSON and CSV rows as pages arrive.
- Use backend artifact export for very large jobs.
- Add `--max-items`.
- Add memory-conscious writer interfaces.

### 20.3 Secret Leakage

Risk:

- Debug logs or errors could reveal API keys.

Mitigation:

- Central redaction package.
- Tests for redaction.
- Never echo auth headers.

### 20.4 Agent Misuse

Risk:

- Agents may call destructive commands or paste huge outputs into chat.

Mitigation:

- Dry-run default for risky commands.
- Skills guidance.
- Large-output warnings.
- Structured hints recommending `--output`.

### 20.5 Cross-Platform Credential Storage

Risk:

- Linux keychain behavior varies.

Mitigation:

- Provider chain with env var fallback.
- Clear `doctor auth` diagnostics.
- Document CI usage separately.

### 20.6 Export Endpoint Delays

Risk:

- CLI blocks waiting for long exports.

Mitigation:

- Async export jobs.
- `--wait` optional.
- Polling status commands.
- Timeout and retry controls.

## 21. Initial Backlog

### 21.1 Foundation

- Create Go module.
- Add Cobra root.
- Add global flags.
- Add output package.
- Add error package.
- Add config package.
- Add profile package.
- Add credential interfaces.
- Add HTTP client.
- Add version command.
- Add CI workflow.

### 21.2 Auth

- Implement API key import.
- Implement keychain provider.
- Implement env provider.
- Implement auth status.
- Implement whoami.
- Implement API key list/create/revoke.
- Implement doctor auth.

### 21.3 Open API

- Inventory existing backend endpoints.
- Define Open API stable endpoint list.
- Add schema endpoint.
- Add export endpoint.
- Normalize pagination where needed.
- Normalize error shape where needed.

### 21.4 Read Commands

- Implement `search trademarks`.
- Implement `search ttab-cases`.
- Implement `search cases`.
- Implement `search office-action-documents`.
- Implement `portfolio summary`.
- Implement `portfolio trademarks list`.
- Implement `portfolio office-actions list`.
- Implement `portfolio conflict-actions list`.
- Implement `portfolio cbp-recordations list`.
- Implement `competitor activities list`.

### 21.5 Export

- Implement CSV writer.
- Implement NDJSON writer.
- Implement output path validation.
- Implement streaming page writer.
- Implement export create/status/download.
- Implement artifact metadata output.

### 21.6 Agent

- Write `tmc-shared` skill.
- Write `tmc-search` skill.
- Write `tmc-portfolio` skill.
- Write `tmc-reports` skill.
- Test with external agents.

### 21.7 Release

- Add goreleaser config.
- Add npm wrapper.
- Add checksum generation.
- Add install script.
- Add update command.
- Add release docs.

## 22. Example Agent Workflows

### 22.1 Large Trademark Search

User request:

```text
Export all live Apple trademarks in class 9.
```

Agent should run:

```bash
tmc search trademarks --owner "Apple Inc." --status live --class 9 --page-all --format csv --output apple-class-9-live.csv
```

Agent final response should summarize:

- File path.
- Row count.
- Top few observations.
- Any warnings.

Agent should not paste the full CSV.

### 22.2 Portfolio Status Count

User request:

```text
How many marks in my portfolio are pending vs registered?
```

Agent should run:

```bash
tmc portfolio status-counts --format json
```

This should return a small inline answer.

### 22.3 Competitor Activity Export

User request:

```text
Give me all high importance Nike competitor activities this quarter in a file.
```

Agent should run:

```bash
tmc competitor activities list --competitor-name Nike --importance high --page-all --format ndjson --output nike-high-activities.ndjson
```

### 22.4 Raw API Fallback

User request references a backend endpoint not yet wrapped by CLI.

Agent can run:

```bash
tmc api GET /api/v1/some/new/resource --params '{"page":1,"page_size":20}'
```

## 23. Documentation Deliverables

Required docs:

- `README.md`
- `docs/install.md`
- `docs/auth.md`
- `docs/config.md`
- `docs/commands.md`
- `docs/output-contract.md`
- `docs/error-contract.md`
- `docs/agent-usage.md`
- `docs/examples.md`
- `docs/release.md`
- `docs/security.md`

Generated docs:

- Command reference from Cobra.
- JSON schema reference from embedded command schemas.
- Markdown examples from E2E fixtures.

## 24. Final Recommendation

Build TMCopilot CLI as a durable Open API client with agent-grade output and error contracts. Prioritize complete architecture and contracts first, then implement commands in a phased order that maximizes immediate value:

1. Auth, config, doctor.
2. Large read/list commands.
3. File output and export/artifact workflows.
4. Raw API and schema.
5. Competitor, gap analysis, and reports.
6. MCP bridge and Agent Skills.
7. Write commands with dry-run and confirmation.
8. Distribution and production hardening.

This produces a CLI that solves the MCP context-length problem while also becoming a reliable scripting and automation interface for TMCopilot.
