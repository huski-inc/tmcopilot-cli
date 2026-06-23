# Catalog Reference

The catalog is generated from `tmcopilot-project/backend/docs/swagger/swagger.json`.

Coverage values:

- `typed`: has a first-class CLI command.
- `raw-ready`: generated endpoint metadata that is not exposed through the public CLI catalog.
- `raw`: generated endpoint metadata that is not exposed through the public CLI catalog.

Only public typed endpoints are exposed through CLI catalog, endpoint/schema inspection, or raw API fallback.

Useful commands:

```bash
tmc api catalog --coverage typed
tmc api catalog --tag trademark
tmc api endpoint GET /auth/me
make openapi-check
```
