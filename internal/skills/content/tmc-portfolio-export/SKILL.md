---
name: tmc-portfolio-export
version: 1.1.0
description: "Portfolio list, monitored trademarks, tasks, competitor lists, and large-result export patterns."
cliHelp: "tmc portfolio --help"
---

# tmc-portfolio-export

Use this skill for portfolio trademarks, action lists, tasks, competitors, and large exports.

## Portfolio Commands

```bash
tmc portfolio trademarks list --keyword nike --country US --class 25
tmc portfolio trademarks get <trademark-id>
tmc portfolio trademarks monitored --monitor-type 1
tmc portfolio counts
tmc portfolio actions office --keyword nike
tmc portfolio actions conflict --risk high
tmc portfolio activity list --keyword nike
tmc portfolio tasks list --status 1
tmc portfolio tasks get <task-id>
```

## Competitor Commands

```bash
tmc competitors list --search nike
tmc competitors activities list --competitor-id <id>
tmc competitors reports list
```

## Large Export Pattern

```bash
tmc portfolio trademarks list \
  --page-all \
  --page-size 100 \
  --max-rows 10000 \
  --format ndjson \
  --fields id,mark,serial_number,status \
  --output trademarks.ndjson \
  --manifest trademarks.manifest.json
```

Use `--progress` when running a long export from a terminal or automation log.
