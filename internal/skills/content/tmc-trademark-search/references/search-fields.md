# Search Fields

`tmc search trademarks` accepts comma-separated or repeated array flags.

Examples:

```bash
tmc search trademarks --class 25,35
tmc search trademarks --class 25 --class 35
tmc search trademarks --owner "Nike Inc" --owner "Nike Innovate C.V."
```

The command maps these flags into Open API request arrays:

```json
{
  "class": ["25", "35"],
  "owners": ["Nike Inc", "Nike Innovate C.V."]
}
```
