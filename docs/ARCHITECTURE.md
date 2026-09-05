# Architecture

Two sections. **Target** is where the project is going and changes
rarely. **Current state** is where it is and is updated as changes
land — if the two sections disagree about what exists, current state
wins.

## Target

```
                 ┌───────────────────── procdash (one binary, native on host) ─────────────────────┐
                 │                                                                                 │
  /proc ───────▶ │  internal/procfs ──▶ sampler (rate engine) ──▶ JSON API  /api/processes         │
  (host kernel)  │   (parsers)      └─▶ collector ───────────────▶ exporter /metrics               │
                 │                                                                                 │
                 └──────────────────────────┬───────────────────────────┬──────────────────────────┘
                                            │ scrape                    │ Infinity datasource
                                            ▼                          ▼
                                       Prometheus ────────────────▶ Grafana (dashboards as code)
                                            │
                                            ▼
                                      Alertmanager ──▶ notifications

  MCP server ── read-only tools over the JSON API + PromQL (a consumer, not a third plane)
```

### The core decision: two planes, split by data shape

Numeric time series (CPU seconds, bytes, counts) go through
Prometheus — history, `rate()`, and alerting are the point.
Point-in-time rich detail (FD tables, connection tuples, thread
lists) is unbounded-cardinality data that must never enter metric
labels; it goes through a JSON API rendered by Grafana's Infinity
datasource. See [decisions.md](decisions.md) ADR-002.

### Components (target)

| Component | Role |
|---|---|
| `internal/procfs` | `/proc` text → typed structs. Stdlib-only, pure parse core, injectable filesystem root, never logs. |
| sampler | Process discovery + snapshot diffing. The single rate engine; the API reuses it. |
| collector | Custom Prometheus collector over procfs structs. Converts kernel units to base units; labels bounded by watch-list/aggregation config. |
| api | The JSON detail plane: `/api/processes`, `/api/process/{pid}`. |
| httpx (or equivalent) | The single HTTP construction point: every server gets the fixed middleware chain (recovery → logging → metrics → timeout). |
| `cmd/procdash` | Flags, config, assembly, presentation. No logic. |
| `deploy/` | docker-compose for Prometheus, Grafana, Alertmanager; datasources, dashboards, and alert rules provisioned from repo files. |
| MCP server | Thin adapter exposing typed, read-only tools over the API and PromQL. |

### Rules that shape the dependency graph

- Direction: `cmd` → service packages (collector, api, sampler) →
  foundation (`procfs`, `httpx`) → stdlib. Never the reverse.
- `internal/procfs` never learns what a terminal, a metric, or a JSON
  document is. Data flows up through return values.
- The exporter runs natively (it needs the real `/proc`); everything
  stateless runs in containers and is reproducible from the repo with
  `docker compose up`.

## Current state (2026-09-05)

`internal/procfs` exists and is the only package:

- `FS`/`Proc` types with an injectable root (`NewFS`) — the test seam
  every parser uses.
- Error classification in one place (`readFile`): vanished process →
  `ErrGone`; permission denied → wrapped `fs.ErrPermission`; anything
  else wrapped with path context. Parse failures are loud and name
  the offending line.
- `/proc/<pid>/status` parser: identity, state, Uid/Gid, threads, and
  the `Vm*` memory block with explicit absence modeling (`HasVM`) for
  kernel threads and zombies.
- Fixture harness under `testdata/` (live-captured kernel output) +
  table-driven tests; parse functions at 100% statement coverage.

No binary is built yet; `go.mod` has zero dependencies. Next:
remaining snapshot parsers (`stat`, `io`, `uptime`), then the
one-shot CLI.
