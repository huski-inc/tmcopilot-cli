# TMCopilot CLI

`tmc` is the command-line tool for TMCopilot. It lets human users and AI agents query, export, and operate on TMCopilot data directly from the terminal.

The CLI can be invoked as either `tmc` or `tmcopilot`. `tmc` is the short command used in most examples; `tmcopilot` is provided as a semantic alias for agents and users who refer to the product by name.

Use it when you need to:

- Search trademarks, owners, lawyers, Office Actions, TTAB cases, and lawsuits
- Export portfolio trademarks, monitoring results, and competitor intelligence
- Create, run, and download Gap Analysis results
- Let Claude Code, Codex, Cursor, and other agents call TMCopilot through stable commands
- Use an API key from scripts or CI jobs

Install · Quick Start · Authentication · Features · Agent Skills · Output & Export · Safety

## Why tmc?

- Ready to use: install the native binary with one `npx` command on macOS, Linux, or Windows
- Browser authorization: `tmc setup` opens the authorization page and stores a CLI API key after approval; agents can use `--no-wait` and resume later
- Agent friendly: built-in `skills` docs help agents learn conventions before choosing commands
- Structured output: JSON by default, with `pretty`, `raw`, `ndjson`, and `csv` for people and programs
- Large exports: paginated commands support `--page-all`, field selection, file output, and manifests
- Safer operations: API keys are never printed; writes support `--dry-run`; destructive commands require `--yes`
- Automatic updates: interactive terminals check every two hours and install newer CLI releases automatically
- Raw API fallback: use `tmc api` only for public catalog endpoint debugging

## Features

| Area | What you can do |
| --- | --- |
| Trademark search | Search US trademarks, view details, generate summaries, and get search tips |
| Owners and companies | Search trademark owners / companies and view owner rankings |
| Lawyers and attorneys | Search lawyers / attorneys, rankings, and contact information |
| Office Actions | Search Office Actions by mark, issue type, and related filters |
| TTAB | Search TTAB cases and fetch cases by case number |
| Lawsuits | Search lawsuit wide-table data and fetch lawsuit details by case number |
| Common law and domains | Search app stores, ecommerce/social handles, web evidence, and domain names |
| Trademark image search | Create image search tasks, inspect results, and download USPTO Office Action documents |
| Portfolio | View portfolio trademarks, monitoring summaries, counts, and activity |
| Portfolio Actions | Query Office Action, conflict, CBP, and other action lists |
| Competitors | Query competitors, competitor activities, and reports |
| Gap Analysis | Create, run, wait for, inspect, report on, and share gap analyses |
| Files | Call file listing, presign, and upload APIs |
| API / Schema | Call REST APIs directly and inspect endpoint schemas and the built-in catalog |
| Agent Skills | Read embedded agent guidance and domain command references |

## Install And Quick Start

### Requirements

Before you start, make sure you have:

- Node.js 16+ with `npx`
- A TMCopilot account you can sign in to

Go is only required when building from source.

### Quick Start For Human Users

Step 1: install the CLI.

```bash
npx @tmcopilot/cli@latest install
tmcopilot version
tmc version
```

The npm installer writes `tmc` and `tmcopilot` to the npm global `bin` directory or another common CLI install directory such as `/opt/homebrew/bin`, `/usr/local/bin`, or `~/.local/bin`. Set `TMC_INSTALL_DIR` when you want a specific install location:

```bash
TMC_INSTALL_DIR="$HOME/.local/bin" npx @tmcopilot/cli@latest install
```

macOS and Linux users can also install with the shell script:

```bash
curl -fsSL https://raw.githubusercontent.com/huski-inc/tmcopilot-cli/main/scripts/install.sh | sh
```

Step 2: authorize the CLI.

```bash
tmc setup
```

The command opens a browser authorization page. After you sign in and approve access, the CLI polls for the one-time API key and stores it locally. The raw API key is never printed to the terminal.

If the terminal cannot open a browser automatically:

```bash
tmc setup --no-browser
```

Open the printed authorization URL in your browser, approve the request, and leave the command running until it finishes.

Step 3: verify authentication.

```bash
tmc auth status --check
tmc auth whoami
```

Step 4: start using TMCopilot.

```bash
tmc search trademarks --name Nike --class 25,35 --limit 20
tmc portfolio trademarks list --page 1 --page-size 20
```

The CLI checks for updates automatically at most once every two hours in interactive terminals. When a newer version is available, it runs the npm installer automatically and writes installer output to stderr.

### Quick Start For AI Agents

> These steps are intended for Claude Code, Codex, Cursor, and similar agents. The browser authorization step still requires the user to approve access.

Step 1: install the CLI.

```bash
npx @tmcopilot/cli@latest install
```

Step 2: read the shared conventions.

```bash
tmc agent bootstrap
tmc skills list
tmc skills read tmc-shared
```

Step 3: create a non-blocking browser authorization request.

```bash
tmc setup --no-wait
```

The command prints an authorization URL and a request ID, stores the local polling token under `~/.tmcopilot`, and exits. The raw API key and poll token are not printed.

Step 4: ask the user to open the authorization URL and approve access, then resume the request.

```bash
tmc setup --request-id <request_id>
```

Step 5: verify access.

```bash
tmc agent bootstrap --check
tmc auth status --check
tmc auth workspaces
```

Step 6: read the domain skill for the task.

```bash
tmc skills read tmc-trademark-search
tmc skills read tmc-portfolio-export
tmc skills read tmc-gap-analysis
```

## Authentication

| Command | Description |
| --- | --- |
| `tmc setup` | Recommended entry point. Authorizes in the browser and stores a local API key |
| `tmc setup --no-browser` | Prints the authorization URL instead of opening a browser |
| `tmc setup --no-wait` | Creates an authorization request, prints the URL and request ID, stores the local polling token, and exits |
| `tmc setup --request-id <request_id>` | Resumes a pending `--no-wait` authorization after the user approves access |
| `tmc auth login` | Same browser authorization flow, useful for reauthorizing the active profile |
| `tmc auth login --no-wait` | Non-blocking authorization flow for agent environments |
| `tmc auth login --request-id <request_id>` | Resume a pending non-blocking authorization request |
| `tmc auth status --check` | Shows local auth state and verifies credentials with `/auth/me` |
| `tmc auth whoami` | Shows the current authenticated user |
| `tmc auth workspaces` | Lists accessible workspaces |
| `tmc auth logout` | Removes local credentials for the active profile |
| `tmc auth import-key --api-key-stdin` | Imports an existing API key for scripts or CI |
| `tmc auth api-keys list` | Lists API keys for the current account |
| `tmc auth api-keys create --name <name>` | Creates an API key |
| `tmc --yes auth api-keys revoke <id>` | Revokes an API key |

If a script or CI job already has an API key, import it directly:

```bash
printf '%s' "$TMCOPILOT_API_KEY" | tmc setup --api-key-stdin
```

You can also use an environment variable to override local credentials for the current shell:

```bash
export TMCOPILOT_API_KEY=tmc_...
tmc auth whoami
```

Supported environment variables:

- `TMCOPILOT_API_KEY` or `TMC_API_KEY`
- `TMCOPILOT_ENDPOINT` or `TMC_ENDPOINT`
- `TMCOPILOT_HOME`, which changes the local config directory from the default `~/.tmcopilot`
- `TMCOPILOT_NO_UPDATE_CHECK=1`, which disables automatic update checks
- `TMCOPILOT_NO_AUTO_UPDATE=1`, which keeps automatic checks but prevents interactive auto-install and prints the install command instead
- `TMCOPILOT_UPDATE_CHECK_INTERVAL=2h`, which changes the automatic update check interval

## Updates

`tmc` automatically checks npm package metadata at most once every two hours. In interactive terminals, when a newer version is available, it runs `npx --yes @tmcopilot/cli@<channel> update` and keeps all installer output on stderr so command stdout stays machine-readable. In non-interactive scripts and agent runs, the check is lightweight: it never installs automatically, and it writes only an update notice plus install command to stderr when an update is available. The check is silent when there is no newer version, when the network is unavailable, or when the binary is a local `dev` build.

Check manually without installing:

```bash
tmc update check
```

Check and install from the current channel:

```bash
tmc update
```

Install the latest stable release explicitly:

```bash
npx --yes @tmcopilot/cli@latest update
```

Install the latest experimental release explicitly:

```bash
npx --yes @tmcopilot/cli@experimental update
```

Uninstall persistent CLI commands:

```bash
tmc uninstall --dry-run
tmc uninstall --yes
npx --yes @tmcopilot/cli@latest uninstall
```

`tmc uninstall` removes the local `tmc` and `tmcopilot` binaries from the current install directory and keeps config and credentials by default. Add `--remove-config` only when you also want to delete the local `~/.tmcopilot` config directory.

The update check cache is stored at `~/.tmcopilot/update-check.json`.

## Common Commands

### Trademark Search

```bash
tmc search trademarks --name Nike --class 25,35 --limit 20
tmc search detail 97346091 --country US
tmc search office-actions --mark Nike --issue-type likelihood_confusion
tmc ttab search --plaintiff Nike --issue opposition
tmc ttab case <case-number>
tmc lawsuits search --party Nike --trademark AIR --limit 20
tmc lawsuits get <case-number>
tmc search owners --name "Nike"
tmc lawyers search --name Smith --state CA --limit 20
tmc lawyers get <graph-id>
tmc lawyers trademarks <graph-id> --limit 20
tmc lawyers law-firms <graph-id> --sort-name asc
tmc search image create --bucket tmc-images --key uploads/mark.png --country US,CA
tmc search image result <task-id>
tmc --output office-action.pdf search uspto-document --serial-number 97346091 --document-page-id <id> --document-type <type> --document-date <date>
tmc search summary --data @summary-request.json
```

Typed trademark search sends `["Exact","Fuzzy","Phonetic"]` when `--similarity` is not provided. Use repeated or comma-separated `--similarity` values to narrow the analysis types.

### Common Law And Domains

```bash
tmc common-law search app-store --name Nike --platform ios
tmc common-law search social-handle --name Nike --platform instagram
tmc common-law search google-text --name Nike
tmc common-law max-similarity --keyword Nike
tmc domain search --keyword nike --limit 20
tmc domain max-similarity --keyword nike
```

### Portfolio

```bash
tmc portfolio trademarks list --keyword nike --country US --class 25
tmc portfolio trademarks get <trademark-id>
tmc portfolio trademarks import-preview --owner-name Nike --country US
tmc portfolio trademarks import --owner-name Nike --country US
tmc portfolio trademarks update <trademark-id> --text "NIKE" --status 10
tmc portfolio trademarks metadata get <trademark-id>
tmc portfolio trademarks metadata update <trademark-id> --owner-name "Nike Inc." --nice-class 25,35
tmc portfolio trademarks monitor update <trademark-id> --office-action-enable=true --conflict-action-enable=false
tmc portfolio trademarks monitor batch-toggle --trademark-id <id1>,<id2> --monitor-type conflict --enable=true --conflict-mode text
tmc portfolio groups list --keyword nike
tmc portfolio groups monitor-toggle <group-id> --monitor-type office_action --enable=true
tmc portfolio monitored-summary
tmc portfolio counts
tmc portfolio actions office list --keyword nike --status 1
tmc portfolio actions office deadlines --limit 10
tmc portfolio actions office for-trademark <trademark-id>
tmc portfolio actions office status <trademark-id> <action-id> --status 20 --note "Reviewed"
tmc portfolio actions conflict list --risk high --sort due_date --sort-dir asc
tmc portfolio actions conflict groups --risk high --group-by mark
tmc portfolio actions conflict for-trademark <trademark-id>
tmc portfolio actions conflict status <trademark-id> <action-id> --status 20
tmc portfolio actions cbp list --status active
tmc portfolio actions cbp service-requests
tmc portfolio actions cbp submit --request-type renew --trademark-id <trademark-id>
```

### Competitors

```bash
tmc competitors list --search nike --market US --importance high
tmc competitors activities list --competitor-id <id> --market US
tmc competitors reports list --page 1 --page-size 20
```

### Gap Analysis

```bash
tmc gap list --search nike --status completed
tmc gap create --title "Nike vs Adidas" --base-company-name Nike --benchmark-company-name Adidas --run-immediately
tmc gap wait <id> --poll-interval 5s --wait-timeout 10m
tmc gap results <id>
tmc gap generate-report <id> --selected-class 25,35
tmc gap shares create <id>
```

### Files

```bash
tmc files list
tmc files presign --data @file-presign.json
tmc files upload-presign --data @upload-presign.json
```

## Three Command Layers

`tmc` offers three levels of command access, from routine workflows to fully custom REST API calls.

### 1. Typed Commands

Common workflows are available as stable commands:

```bash
tmc search trademarks --name Nike --limit 20
tmc common-law search social-handle --name Nike --platform instagram
tmc domain search --keyword nike --limit 20
tmc portfolio trademarks list --page-all --format ndjson --output trademarks.ndjson
tmc gap create --data @gap-create.json
```

Run `tmc <command> --help` to inspect subcommands and flags.

### 2. Schema / Catalog

When you are not sure which parameters a command accepts, inspect its schema first:

```bash
tmc schema search trademarks
tmc api catalog --coverage typed
tmc api endpoint GET /auth/me
tmc api schema POST /trademark/search
```

`tmc schema <command...>` includes agent-oriented metadata:

- `safety`: whether the command is read-only, has side effects, is destructive, supports `--dry-run`, or requires `--yes`
- `pagination`: whether `--page-all`, `--fields`, and `--manifest` are supported
- `examples`: command-specific next commands to inspect or dry-run

### 3. Raw API

Use raw API calls only for public catalog endpoint debugging or reproducing typed command requests:

```bash
tmc api GET /auth/me
tmc api POST /trademark/search --data @request.json
tmc api endpoint POST /trademark/search
```

Only public catalog endpoints are available through `tmc api catalog`, `tmc api endpoint`, `tmc api schema`, and the raw `tmc api` fallback.

## Output And Export

The default output is a JSON envelope:

```json
{"ok":true,"data":{},"meta":{"status_code":200}}
```

Errors are written to stderr:

```json
{"ok":false,"type":"auth_error","message":"http 401: unauthorized","trace_id":"..."}
```

Output formats:

| Format | Description |
| --- | --- |
| `json` | Default compact JSON envelope |
| `pretty` | Formatted JSON for human reading |
| `raw` | API data only, without the CLI envelope |
| `ndjson` | Newline-delimited JSON for large exports |
| `csv` | CSV export for spreadsheet tools |

Examples:

```bash
tmc --format pretty auth status
tmc --format raw auth whoami
tmc --output result.json search trademarks --name Nike
```

### Paginated Exports

Paginated list commands support:

```bash
--page 1
--page-size 100
--page-all
--max-pages 10
--max-rows 10000
--fields id,mark,serial_number,status
--output export.ndjson
--manifest export.manifest.json
--progress
```

Example portfolio export:

```bash
tmc portfolio trademarks list \
  --page-all \
  --page-size 100 \
  --format ndjson \
  --fields id,mark,serial_number,status \
  --output portfolio.ndjson \
  --manifest portfolio.manifest.json
```

`--page-all` requests one page at a time. It does not ask the backend to return every row in a single response.

## Agent Skills

The CLI embeds agent-readable usage guidance. Agents should read `tmc-shared` first, then the domain skill for the task.

| Skill | Description |
| --- | --- |
| `tmc-shared` | Global authentication, output, safety, pagination, and command-selection rules |
| `tmc-trademark-search` | Trademark, owner, lawyer, Office Action, TTAB, and lawsuit search |
| `tmc-portfolio-export` | Portfolio lists, pagination, field selection, and export |
| `tmc-gap-analysis` | Gap Analysis creation, execution, waiting, results, and reports |
| `tmc-openapi` | API catalog, schema, and raw API usage |

```bash
tmc agent bootstrap
tmc skills list
tmc skills read tmc-shared
tmc skills read tmc-trademark-search
tmc skills read tmc-openapi/references/catalog.md
```

`tmc agent bootstrap --check` returns one machine-readable snapshot with command aliases, auth status, configured endpoint, available skills, discovery commands, safety guidance, and recommended next steps. Use it as the first command when an AI agent is unsure how the local CLI is configured.

`tmc skills read` follows the global output contract and returns a JSON envelope by default. Use `tmc --format raw skills read <skill>` only when raw markdown is required.

## Configuration And Profiles

The default config directory is `~/.tmcopilot`. Common configuration commands:

```bash
tmc config show
tmc config set endpoint https://api.tmcopilot.ai
tmc config set workspace <workspace-id>
tmc config profile list
tmc config profile add local
tmc config profile use local
tmc --profile local auth status
```

Common global flags:

```bash
tmc --endpoint http://localhost:8080 <command>
tmc --workspace <workspace-id> <command>
tmc --profile local <command>
tmc --timeout 60s <command>
```

## Safety Notes

`tmc` can read and operate on data that your TMCopilot account can access. When using it with an AI agent, keep these risks in mind:

- The agent may misunderstand your intent and run the wrong query or write operation
- Do not paste API keys, authorization URLs, local pending authorization files, or sensitive export files into untrusted environments
- Use `--dry-run --request-out request.json` before write operations to inspect the request
- Delete and revoke operations require `--yes`
- On shared machines, run `tmc auth logout` when you are done

Preview a request:

```bash
tmc --dry-run --request-out request.json api POST /trademark/search --data @request.json
```

Revoke an API key:

```bash
tmc auth api-keys list
tmc --yes auth api-keys revoke <id>
```

## Diagnostics

```bash
tmc doctor
tmc doctor network
tmc doctor auth
tmc auth status --check
```

## Local Development

Most users do not need to build from source. To work on the CLI locally:

```bash
make test
make vet
make build
```

Use a temporary config directory when testing against a local backend:

```bash
export TMCOPILOT_HOME="$(mktemp -d)"
go run . --endpoint http://localhost:8080 setup --no-wait
go run . --endpoint http://localhost:8080 setup --request-id <request_id>
```
