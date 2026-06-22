# Catalog Reference

The catalog is generated from `tmcopilot-project/backend/docs/swagger/swagger.json`.

Coverage values:

- `typed`: has a first-class CLI command.
- `raw-ready`: no typed command yet, but likely suitable for `tmc api`.
- `raw`: available through `tmc api`; inspect endpoint shape before calling.

Useful commands:

```bash
tmc api catalog --coverage typed
tmc api catalog --tag trademark
tmc api endpoint GET /auth/me
make openapi-check
```
