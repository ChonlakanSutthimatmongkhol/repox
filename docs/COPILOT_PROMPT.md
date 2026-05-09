# Repox — Copilot Kickoff Prompt

Use this prompt with GitHub Copilot to start building Phase 1.

---

## Prompt

```
You are building "Repox" — a Go CLI tool that generates feature scaffolds matching a repo's conventions.

Read COPILOT_SKILL.md for full architecture, rules, and implementation checklist.

Build Phase 1 in checkpoint order (1 through 7). After each checkpoint:
1. Ensure `go build ./cmd/repox` succeeds
2. Ensure all tests pass with `go test ./...`
3. Confirm before moving to next checkpoint

Start with Checkpoint 1: Project Setup.

Key rules:
- Go module: github.com/ChonlakanSutthimatmongkhol/repox
- CLI framework: cobra
- All business logic in internal/ — never in cmd/
- Templates use Go text/template with embed.FS
- File writer refuses overwrite without --force
- Every checkpoint must compile and pass tests

Begin.
```

---

## Tips for Using with Copilot

1. **Put COPILOT_SKILL.md in repo root** — Copilot reads it as context
2. **Work checkpoint by checkpoint** — don't let it skip ahead
3. **Verify each checkpoint** before saying "next checkpoint"
4. **If Copilot generates too many files at once**, say: "Stop. Let's verify this checkpoint first."
5. **After Phase 1 is done**, you can create COPILOT_SKILL_PHASE2.md for the scanner phase
