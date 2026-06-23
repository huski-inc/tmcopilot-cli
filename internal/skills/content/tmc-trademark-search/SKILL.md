---
name: tmc-trademark-search
version: 1.3.0
description: "Trademark, office action, TTAB, lawsuit, lawyer, and company/owner search commands for TMCopilot."
cliHelp: "tmc search --help; tmc lawsuits --help; tmc lawyers --help"
---

# tmc-trademark-search

Use this skill when the user asks to search trademarks, office action cases, TTAB cases, lawsuits, lawyers, or companies/owners.

## Commands

| Need | Command |
|---|---|
| Trademark search | `tmc search trademarks` |
| Trademark search alias | `tmc search trademark` |
| Trademark detail | `tmc search detail` |
| Office action / case search | `tmc search office-actions` |
| TTAB case search | `tmc ttab search` or `tmc search ttab` |
| TTAB case detail | `tmc ttab case` or `tmc search ttab-case` |
| Lawsuit search | `tmc lawsuits search` or `tmc search lawsuits` |
| Lawsuit detail | `tmc lawsuits get` or `tmc search lawsuit` |
| Brand owner lawsuits | `tmc lawsuits brand-owner` |
| Lawyer lawsuits | `tmc lawyers lawsuits` or `tmc lawsuits lawyer` |
| Lawyer search | `tmc lawyers search` or `tmc search lawyers` |
| Attorney search alias | `tmc attorneys search` or `tmc search attorneys` |
| Lawyer ranking | `tmc lawyers ranking` or `tmc search lawyer-ranking` |
| Lawyer contact | `tmc lawyers contact` or `tmc search lawyer-contact` |
| Lawyer detail | `tmc lawyers get` |
| Lawyer trademarks | `tmc lawyers trademarks` |
| Lawyer law firms | `tmc lawyers law-firms` |
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
tmc ttab search --plaintiff Nike --issue opposition
tmc ttab case <case-number>
```

Lawsuit search:

```bash
tmc lawsuits search --party Nike --trademark AIR --limit 20
tmc lawsuits get <case-number>
tmc lawsuits brand-owner <graph-id> --limit 20
tmc lawsuits lawyer <graph-id> --sort-case-at desc
```

Lawyer search:

```bash
tmc lawyers search --name Smith --state CA --limit 20
tmc lawyers get <graph-id>
tmc lawyers trademarks <graph-id> --limit 20
tmc lawyers law-firms <graph-id> --sort-name asc
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
tmc schema lawsuits search
tmc schema lawyers search
tmc schema common-law search social-handle
tmc schema domain search
tmc schema search image create
```
