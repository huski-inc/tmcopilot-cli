---
name: tmc-openapi
version: 1.1.2
description: "Use TMCopilot OpenAPI catalog, endpoint inspection, public raw API debugging, request dry-run, and file download."
cliHelp: "tmc api --help"
---

# tmc-openapi

Use this skill when the user asks to inspect available Open API commands or debug public catalog endpoint requests.

## Discover Endpoints

```bash
tmc api catalog --coverage typed
tmc api catalog --search lawsuit
tmc api endpoint POST /trademark/search
tmc api schema POST /trademark/search
```

Only public catalog endpoints are exposed through CLI catalog, endpoint/schema inspection, or raw API fallback.

## Raw API Call

```bash
tmc api GET /auth/me
tmc api POST /trademark/search --data @request.json
```

## Dry Run

```bash
tmc --dry-run --request-out request.json api POST /trademark/search --data @request.json
```

## Download Raw Response

```bash
tmc --output response.bin api download GET <public-download-path>
```
