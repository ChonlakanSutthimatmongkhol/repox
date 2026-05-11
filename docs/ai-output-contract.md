# Repox AI Output Contract

AI-facing commands should keep output compact, deterministic, and markdown-shaped.

Commands with `--ai` use these sections:

```md
## Summary
## Detected Conventions
## Important Findings
## Related Examples
## Suggested Next Commands
## Warnings
```

Rules:

- Avoid raw noisy logs.
- Include suggested next commands.
- Include warnings when data is missing or likely stale.
- Prefer scanned examples and `--like <existing-feature>` before broad generic generation.

Primary AI workflow:

```bash
repox doctor
repox explain --ai
repox map --ai
repox plan feature <name> --ai
repox generate feature <name> --like <existing> --dry-run --ai
```
