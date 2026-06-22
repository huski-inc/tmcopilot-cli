---
name: tmc-gap-analysis
version: 1.1.0
description: "Create, run, poll, inspect, report, and share TMCopilot gap analyses."
cliHelp: "tmc gap --help"
---

# tmc-gap-analysis

Use this skill when the user asks about gap analysis, competitor comparison, report generation, or shared gap reports.

## Commands

```bash
tmc gap list --search nike --status completed
tmc gap create --title "Nike vs Adidas" --base-company-name Nike --benchmark-company-name Adidas --run-immediately
tmc gap get <id>
tmc gap run <id>
tmc gap wait <id> --poll-interval 5s --wait-timeout 10m
tmc gap results <id>
tmc gap reports <id>
tmc gap generate-report <id> --selected-class 25,35
tmc gap shares create <id>
tmc gap shares list <id>
tmc gap shares get <token>
tmc gap shares revoke <id> <token>
```

## Safety

Use `--dry-run --request-out request.json` before create or destructive operations when the user has not confirmed inputs.

Delete/revoke style operations require `--yes`.
