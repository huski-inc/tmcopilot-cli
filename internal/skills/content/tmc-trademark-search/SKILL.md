---
name: tmc-trademark-search
version: 1.1.0
description: "Trademark, office action, TTAB, lawyer, and company/owner search commands for TMCopilot."
cliHelp: "tmc search --help"
---

# tmc-trademark-search

Use this skill when the user asks to search trademarks, office action cases, TTAB cases, lawyers, or companies/owners.

## Commands

| Need | Command |
|---|---|
| Trademark search | `tmc search trademarks` |
| Trademark search alias | `tmc search trademark` |
| Trademark detail | `tmc search detail` |
| Office action / case search | `tmc search office-actions` |
| TTAB case search | `tmc search ttab` |
| TTAB case detail | `tmc search ttab-case` |
| Lawyer search | `tmc search lawyers` |
| Attorney search alias | `tmc search attorneys` |
| Lawyer ranking | `tmc search lawyer-ranking` |
| Lawyer contact | `tmc search lawyer-contact` |
| Company / owner search | `tmc search owners` |
| Company search alias | `tmc search companies` |
| Owner ranking | `tmc search owner-ranking` |
| Trademark image task | `tmc search image create` |
| Trademark image result | `tmc search image result` |
| USPTO Office Action document | `tmc search uspto-document` |
| Common-law app store search | `tmc common-law search app-store` |
| Common-law social handle search | `tmc common-law search social-handle` |
| Common-law web search | `tmc common-law search google-text` |
| Domain name search | `tmc domain search` |

## Examples

Trademark search with multiple classes:

```bash
tmc search trademarks --name Nike --class 25,35,42 --limit 20
```

Equivalent repeated flags:

```bash
tmc search trademarks --name Nike --class 25 --class 35 --class 42
```

Office action search:

```bash
tmc search office-actions --mark Nike --issue-type likelihood_confusion
```

TTAB search:

```bash
tmc search ttab --plaintiff Nike --issue opposition
```

Lawyer search:

```bash
tmc search lawyers --name Smith --state CA --limit 20
```

Company / owner search:

```bash
tmc search owners --name "Nike" --limit 20
```

Common-law and domain evidence:

```bash
tmc common-law search social-handle --name Nike --platform instagram
tmc common-law search google-text --name Nike
tmc domain search --keyword nike --limit 20
```

Trademark image search:

```bash
tmc search image create --bucket tmc-images --key uploads/mark.png --country US,CA
tmc search image result <task-id>
```

USPTO Office Action document download:

```bash
tmc --output office-action.pdf search uspto-document --serial-number 97346091 --document-page-id <id> --document-type <type> --document-date <date>
```

Schema inspection before raw API fallback:

```bash
tmc schema search trademarks
tmc schema search office-actions
tmc schema search ttab
tmc schema common-law search social-handle
tmc schema domain search
tmc schema search image create
```

## Raw API Fallback

If the user asks for lawsuit wide-table search, use raw API until a typed command exists:

```bash
tmc api POST /trademark/wide-table/lawsuits --data @request.json
```
