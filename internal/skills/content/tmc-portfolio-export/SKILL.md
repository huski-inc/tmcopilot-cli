---
name: tmc-portfolio-export
version: 1.1.0
description: "Portfolio list, monitored trademarks, competitor lists, and large-result export patterns."
cliHelp: "tmc portfolio --help"
---

# tmc-portfolio-export

Use this skill for portfolio trademarks, action lists, tasks, competitors, and large exports.

## Portfolio Commands

```bash
tmc portfolio trademarks list --keyword nike --country US --class 25
tmc portfolio trademarks get <trademark-id>
tmc portfolio trademarks monitored --monitor-type 1
tmc portfolio trademarks import-preview --owner-name Nike --country US
tmc portfolio trademarks import --owner-name Nike --country US
tmc portfolio trademarks update <trademark-id> --text "NIKE" --status 10
tmc portfolio trademarks metadata get <trademark-id>
tmc portfolio trademarks metadata update <trademark-id> --owner-name "Nike Inc." --nice-class 25,35
tmc portfolio trademarks monitor update <trademark-id> --office-action-enable=true --conflict-action-enable=false
tmc portfolio trademarks monitor batch-update --trademark-id <id1>,<id2> --office-action-enable=true
tmc portfolio trademarks monitor batch-toggle --trademark-id <id1>,<id2> --monitor-type conflict --enable=true --conflict-mode text
tmc portfolio groups list --keyword nike
tmc portfolio groups monitor-toggle <group-id> --monitor-type office_action --enable=true
tmc portfolio counts
tmc portfolio actions office --keyword nike
tmc portfolio actions conflict --risk high
tmc portfolio activity list --keyword nike
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

For write operations, use `tmc --dry-run --request-out request.json ...` first when the user has not confirmed the exact payload.
