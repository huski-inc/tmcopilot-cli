---
name: tmc-shared
version: 1.3.0
description: "TMCopilot CLI shared guidance: auth, output contracts, safety flags, large-result rules, and when to use catalog/schema-style discovery."
cliHelp: "tmc --help"
---

# tmc-shared

Read this first before using other TMCopilot CLI skills.

## Core Rules

- Use `tmcopilot` or `tmc` for TMCopilot Open API access, not MCP. They are the same CLI; `tmcopilot` is the semantic alias and `tmc` is the short form used in examples.
- Start uncertain sessions with `tmc agent bootstrap` or `tmc agent bootstrap --check` to get command aliases, auth status, skills, safety rules, and recommended next steps.
- Use `tmc --help` and `tmc <command> --help` for command flags.
- Use `tmc schema <command...>` to inspect command flags, endpoint summary, safety metadata, pagination support, and examples before using unfamiliar flags.
- Add `--openapi` only when raw Swagger definitions are necessary.
- Use `tmc api catalog` to discover generated Swagger endpoints.
- Use `tmc api schema METHOD /path` only for raw API fallback or endpoint debugging.
- Use `--output` for large JSON.
- Use `--page-all` only on paginated list commands; it pages through the API one page at a time.
- Prefer `--format ndjson --output file.ndjson --manifest file.manifest.json` for large exports.
- Use `--dry-run --request-out request.json` before write or destructive operations.
- Destructive requests require `--yes`.

## Auth

The default endpoint is `https://api.tmcopilot.ai`. Use `--endpoint http://localhost:8080` only for local development.

```bash
tmc agent bootstrap --check
tmc setup
tmc setup --no-wait
tmc setup --request-id <request_id>
tmc setup --no-browser
printf '%s' "$TMCOPILOT_API_KEY" | tmc setup --api-key-stdin
tmc auth status
tmc auth whoami
tmc auth workspaces
```

`tmc setup` and `tmc auth login` create a browser authorization request, poll for a one-time API key, and store it locally. They do not print the raw key.

For agent environments, prefer `tmc setup --no-wait`. Send the printed authorization URL to the user, wait for approval, then run `tmc setup --request-id <request_id>`. The pending poll token is stored locally and must not be copied into chat or logs.

## Diagnostics

```bash
tmc doctor
tmc doctor --strict=false
tmc api catalog --coverage typed
```

## Output

Default output is a JSON envelope:

```json
{"ok":true,"data":{},"meta":{"status_code":200}}
```

Errors are written to stderr with a stable `type`.
