# Agent Usage Guide

Use `tmcopilot` or `tmc` when an agent needs durable, file-friendly access to TMCopilot Open API results. They are the same CLI; `tmcopilot` is the semantic command alias and `tmc` is the short form used in examples.

Before choosing commands, inspect the embedded skills shipped with the current binary:

```bash
tmc agent bootstrap
tmc skills list
tmc skills read tmc-shared
tmc skills read tmc-trademark-search
```

Use `tmc agent bootstrap --check` when credentials may already be configured. It returns command aliases, auth status, endpoint/profile information, embedded skills, discovery commands, safety guidance, and recommended next steps in one JSON envelope.

Use `tmc skills read <skill>/references/<file>.md --json` when a structured agent response is easier to parse than raw markdown.

Setup order:

- Prefer `tmc setup` for a human-operated terminal.
- The default endpoint is `https://api.tmcopilot.ai`; use `--endpoint http://localhost:8080` only for local development.
- Prefer `tmc setup --no-wait` for agent environments. It creates an authorization request, prints the URL and request ID, stores the local polling token, and exits without printing an API key.
- After the user approves the authorization URL, run `tmc setup --request-id <request_id>` to poll the saved request and store the API key locally.
- Use `tmc setup --no-browser` when the terminal cannot launch a browser but it is acceptable for the command to keep waiting.
- For CI or environments that already have an API key, pipe it into `tmc setup --api-key-stdin`.
- `tmc setup` and `tmc auth login` create an API key authorization request, poll for the one-time API key, store it locally, and do not print the raw key.

Examples:

```bash
tmc --endpoint http://localhost:8080 setup --no-wait
tmc --endpoint http://localhost:8080 setup --request-id <request_id>
```

```bash
printf '%s' "$TMCOPILOT_API_KEY" | tmc setup --api-key-stdin
```

Command selection order:

- Prefer typed commands such as `tmc search trademarks`, `tmc portfolio trademarks list`, and `tmc gap create`.
- Use aliases when they match the user's wording, for example `tmc search companies` for owner/company search and `tmc search attorneys` for lawyer search.
- Use `tmc schema <command...>` to inspect a CLI command's flags, endpoint summary, safety metadata, pagination support, and examples before using an unfamiliar typed command.
- Add `--openapi` only when raw Swagger parameters, responses, and definitions are needed.
- Use `tmc api catalog` to discover endpoints and `tmc api METHOD /path` only when a typed command does not exist.

Prefer CLI when:

- The result may exceed an MCP or chat context window.
- The task needs pagination, repeatability, or CI execution.
- The agent needs a local artifact such as NDJSON, CSV, JSON, or a downloaded file.

Prefer MCP when:

- The agent is discovering capabilities.
- The expected result is small enough to fit directly in context.

Rules for large results:

- Use `--page-all` only with ordinary paginated endpoints; it requests one page at a time.
- Prefer `--output` for large results.
- Prefer `--format ndjson` for agent post-processing.
- Use `--fields` to reduce output width.
- Use `--manifest` when the export must be auditable.

Examples:

```bash
tmc --endpoint http://localhost:8080 portfolio trademarks list \
  --page-all \
  --page-size 100 \
  --format ndjson \
  --fields id,mark,serial_number,status \
  --output trademarks.ndjson \
  --manifest trademarks.manifest.json
```

```bash
tmc search trademarks --name Nike --limit 20 --output search.json
```

```bash
tmc api catalog --coverage raw --tag trademark
tmc schema search trademarks
tmc schema --openapi search trademarks
tmc api schema POST /trademark/max-similarity
tmc api POST /trademark/max-similarity --data @request.json --output result.json
```
