# AGENTS.md

Guidance for AI coding agents contributing to procdash. Deliberately
sparse for now; it grows as the codebase does.

## What this project is

A Go service that parses Linux `/proc` directly and exposes process
observability through two planes: Prometheus metrics (time series)
and a JSON detail API (rich per-process detail). Start with
[README.md](README.md) and [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md);
[docs/runbook.md](docs/runbook.md) says what actually runs today —
do not claim capabilities beyond it.

## Commands

```bash
gofmt -l .        # must print nothing
go vet ./...
go test ./...     # fixture-driven; never requires root
```

## Hard rules

1. `internal/procfs` is **stdlib-only — permanently**. No `gopsutil`,
   no `prometheus/procfs`. Any new dependency anywhere needs a record
   in [docs/decisions.md](docs/decisions.md) with alternatives
   considered.
2. **Parse functions are pure**: bytes in, typed structs out — no
   file I/O, no logging, no `time.Now()` inside parsers. File reading
   is a thin outer layer, and error classification lives in exactly
   one helper (`readFile`) — extend it, never duplicate it.
3. **`internal/procfs` never logs or prints.** It returns values and
   errors; presentation is the caller's job.
4. **Error taxonomy** (ADR-007): vanished process →
   `procfs.ErrGone` (normal — skip or degrade); permission denied →
   wrapped `fs.ErrPermission` (degrade that item, continue);
   malformed content → loud error naming the file. Never collapse
   these classes — callers branch on the distinction.
5. **Units live in field names** (`VmRSSKiB`, `UTimeTicks`,
   `RSSPages`); convert only at display/export boundaries (ADR-006).
6. **All procfs access goes through `FS{root}`** (`NewFS` is the test
   seam). Every parser change ships with fixture-backed, table-driven
   tests under `internal/procfs/testdata/`; capture fixtures from a
   live `/proc` where possible and label hand-crafted ones.
7. Tests assert exact values and cover named negative paths. No
   `time.Sleep` in tests. Root is never required to run the suite.
8. Never read, log, or export `/proc/<pid>/environ`. Anything serving
   HTTP binds localhost unless a decision record says otherwise.
9. Docs are part of the change, in the same commit: run/build
   behavior → [docs/runbook.md](docs/runbook.md); binding choices →
   [docs/decisions.md](docs/decisions.md); investigated surprises →
   [docs/errors.md](docs/errors.md); current-state changes →
   the bottom of [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Where truth lives

| Question | Source |
|---|---|
| What exists right now? | [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md), "Current state" |
| What runs, and how? | [docs/runbook.md](docs/runbook.md) |
| Why is X the way it is? | [docs/decisions.md](docs/decisions.md) |
| Has this failed before? | [docs/errors.md](docs/errors.md) |
