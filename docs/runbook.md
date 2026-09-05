# Runbook

Exact, copy-pasteable commands to build, run, and verify procdash.
Rules of this document:

- Commands are written to be pasted as-is from the repo root.
- If a command here fails as written, that is a bug or documentation
  drift — record it in [errors.md](errors.md).
- Planned capabilities are marked **pending** until they exist.
  Nothing here claims to work before it does.

## Capability index

| Capability | Status |
|---|---|
| `/proc` parsing library (`status`) | **live** |
| Remaining snapshot parsers (`stat`, `io`, `uptime`) | pending |
| Snapshot CLI (`procdash <pid>`) | pending |
| Continuous sampler / top view | pending |
| Prometheus exporter (`/metrics`) | pending |
| Prometheus + Grafana compose stack | pending |
| Deep inspection (threads/FDs/sockets) | pending |
| JSON detail API | pending |
| Host-level metrics | pending |
| Alerting | pending |
| MCP tools | pending |

## Works today

```bash
go version        # need >= 1.25 (see go.mod)

gofmt -l .        # formatting check (empty output = clean)
go vet ./...      # static checks
go test ./...     # full suite: fixture-driven, no root required
go test ./internal/procfs -run TestName   # single test
go test -cover ./...                      # coverage summary
```

The test suite never needs root and touches no live kernel state —
parsers are exercised against committed fixtures under
`internal/procfs/testdata/`.

## Troubleshooting

1. Check [errors.md](errors.md) — investigated problems live there
   with root causes and fixes.
2. If it's new and surprising, it becomes an errors.md entry once
   solved.
