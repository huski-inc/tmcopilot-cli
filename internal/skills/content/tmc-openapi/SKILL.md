---
name: tmc-openapi
version: 1.1.0
description: "Use TMCopilot OpenAPI catalog, endpoint inspection, raw API fallback, request dry-run, and file download."
cliHelp: "tmc api --help"
---

# tmc-openapi

Use this skill when a typed command does not exist or when the user asks to inspect available Open API commands.

## Discover Endpoints

```bash
tmc api catalog --coverage typed
tmc api catalog --coverage raw --tag trademark
tmc api catalog --search lawsuit
tmc api endpoint POST /trademark/search
tmc api schema POST /trademark/search
```

## Raw API Call

```bash
tmc api GET /auth/me
tmc api POST /trademark/search --data @request.json
tmc api POST /trademark/max-similarity --data @request.json --output result.json
```

## Dry Run

```bash
tmc --dry-run --request-out request.json api POST /trademark/search --data @request.json
```

## Download Raw Response

```bash
tmc --output response.bin api download GET /files/raw
```
