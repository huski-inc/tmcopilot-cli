# TMCopilot CLI Local Integration Test Report

Date: 2026-06-18

## Environment

- CLI repo: `/Users/huski/Huski/tmcopilot-cli`
- Backend repo: `/Users/huski/Huski/tmcopilot-project`
- Backend API: `http://localhost:8080`
- Running backend process: `/Users/huski/Huski/tmcopilot-project/.tmcopilot/local-dev/backend-api`
- Local dependencies:
  - `tmcopilot-project-postgres-1`
  - `tmcopilot-project-redis-1`
  - `tmcopilot-project-mysql-1`

## Authentication Setup

A temporary local API key was inserted into the local development Postgres database for user `id=1`.

- Key name: `tmc-cli-local-integration`
- Raw key handling: stored only in a temporary local file during testing
- Cleanup: key revoked after testing; temporary key file deleted

No raw API key is stored in this report.

## Verified CLI Areas

### Core

- `tmc version`
- `tmc api GET /auth/me`
- `tmc config init`
- `tmc auth import-key`
- `tmc auth status --check`
- `tmc auth whoami`
- `tmc auth workspaces`
- `tmc auth api-keys list`
- `tmc doctor auth`
- `tmc doctor network`

Result: passed.

### API Key Write Flow

- `tmc auth api-keys create --name tmc-cli-e2e-revoke`
- `tmc auth api-keys revoke <id>`

Result: passed. The created API key was revoked.

### Portfolio

- `tmc portfolio counts`
- `tmc portfolio monitored-summary`
- `tmc portfolio trademarks list --page 1 --page-size 5`
- `tmc portfolio trademarks monitored --page 1 --page-size 5`
- `tmc portfolio actions office --page 1 --page-size 5`
- `tmc portfolio actions conflict --page 1 --page-size 5`
- `tmc portfolio actions cbp --page 1 --page-size 5`
- `tmc portfolio activity list --page 1 --page-size 5`

Result: passed.

### Streaming Pagination

Verified against real backend data using `portfolio activity`.

- `tmc --format ndjson --output activity.ndjson portfolio activity list --page-all --page-size 1 --max-pages 2 --fields id,category,action`
- `tmc --format csv --output activity.csv portfolio activity list --page-all --page-size 1 --max-pages 2 --fields id,category,action`

Observed result:

- NDJSON export wrote 2 data rows.
- CSV export wrote 1 header row and 2 data rows.
- CLI stdout returned export summary with `pages=2`, `rows=2`, `total=2`, `total_pages=2`.

Result: passed.

### Competitors

- `tmc competitors list --page 1 --page-size 5`
- `tmc competitors activities list --page 1 --page-size 5`
- `tmc competitors reports list --page 1 --page-size 5`

Result: passed.

### Search

- `tmc search tips --owner-name Nike --region us`
- `tmc search owners --name Nike --limit 1`
- `tmc search lawyers --name Smith --limit 1`
- `tmc search trademarks --name Nike --limit 1 --region US`
- `tmc search ttab --plaintiff Nike`
- `tmc search office-actions --mark Nike`

Result: passed.

### Gap Analysis

Initial command failed with invalid source type:

- `base_source_type=owner`
- backend returned `invalid base source type`

Corrected command used backend-supported values:

- `base_source_type=manual_owner_aliases`
- `benchmark_source_type=manual_owner_aliases`

Verified commands:

- `tmc gap create ...`
- `tmc gap get <id>`
- `tmc gap reports <id>`
- `tmc gap delete <id>`

Result: passed. The created gap analysis was archived by the backend delete API.

## CLI Fixes From Integration

### `portfolio trademarks monitored`

The existing backend Swagger and actual response shape for `/portfolio/trademarks/monitored` return `data` as an array, not the unified pagination object.

Fix applied:

- Removed this command from the unified `newPagedListCommand` path.
- Kept `--page`, `--page-size`, `--monitor-type`, and `--param`.
- Removed inherited `--page-all` behavior for this endpoint.

This avoids attempting to stream an array response as a paginated `{items,total,page,page_size,total_pages}` object.

## Final Cleanup

- Temporary local API key `tmc-cli-local-integration` revoked.
- Temporary API key file deleted.
- API key created by CLI write-flow test revoked.
- Gap analysis created by the write-flow test archived through the backend delete API.

## Verification Commands

Final local checks:

```bash
make test
make vet
make build
```
