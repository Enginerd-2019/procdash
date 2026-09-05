# CLAUDE.md

Instructions for Claude Code sessions in this repository. The full
contract for AI coding agents lives in [AGENTS.md](AGENTS.md) and is
binding for all tools — read it before changing code. This file adds
only quick context.

- Build/verify from the repo root (the Go module root):
  `gofmt -l . && go vet ./... && go test ./...`
- The parser rules (AGENTS.md, Hard rules 1–6) are non-negotiable:
  stdlib-only, pure parse functions, one error-classification point,
  units in field names, everything through the injectable `FS` root
  with fixture-backed tests.
- [docs/runbook.md](docs/runbook.md) defines what currently works —
  never report capabilities beyond it.
- Update the relevant file under `docs/` in the same commit as any
  behavior change (AGENTS.md, rule 9).
