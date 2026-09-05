# procdash

**procdash** is a Linux process observability service written in Go.
It reads the kernel's `/proc` interface directly — no shelling out to
`ps`, no third-party parsing libraries — and serves what it finds
through two planes:

- **Metrics** (Prometheus): CPU seconds, memory, disk I/O,
  thread/FD/connection counts — raw cumulative counters in base
  units, exported for `rate()` and alerting to do their jobs.
- **Detail** (JSON API, rendered in Grafana via the Infinity
  datasource): the data that does not belong in metric labels — the
  actual FD table, connection tuples, per-thread state — one click
  away from any metric panel.

One binary serves both planes. Prometheus and Grafana run in
containers, provisioned entirely from files in this repository; the
exporter runs natively because it needs the real `/proc`.

> **Status: early development.** The parsing foundation
> (`internal/procfs`) is taking shape; the CLI and service layers land
> incrementally. [docs/runbook.md](docs/runbook.md) tracks exactly
> what runs today and never claims more.

## How this differs from pinspect

[pinspect](https://github.com/Enginerd-2019/pinspect) is my earlier C
project in the same domain: a dependency-free, one-shot CLI that
parses `/proc` and prints a detailed point-in-time report of a single
process. Its job ends when the report prints.

procdash asks a different question about the same data:

| | pinspect | procdash |
|---|---|---|
| Language | C | Go |
| Shape | one-shot CLI | long-running service (CLI included) |
| Question | "what is this process doing *right now*?" | "what has this process been doing, is it trending worse, and should someone be told?" |
| Output | terminal report | Prometheus metrics, Grafana dashboards, JSON API, alerts |
| History | none — run it again | time-series database |
| Operation | invoked by a human | scraped unattended, 24/7, observing itself |

pinspect proves the parsing; procdash turns the domain into an
*operated* system: continuous collection, rates derived by the TSDB
from raw counters, dashboards and alert rules living in the repo as
code, and the service watching its own resource cost on its own
dashboard.

## Quick start (today)

```bash
go test ./...   # fixture-driven; needs no root and no live dependencies
```

The one-shot snapshot CLI (`procdash <pid>`) is the first runnable
milestone — see the roadmap below and
[docs/runbook.md](docs/runbook.md) for current commands.

## Roadmap

- [ ] **Snapshot CLI** — status/stat/io/uptime parsers + a one-shot
  process report *(parsers landing now)*
- [ ] **Continuous sampler** — process discovery, counter deltas, a
  terminal top-N view
- [ ] **Prometheus exporter** — `/metrics`, custom collector,
  watch-list configuration, cardinality budget
- [ ] **Prometheus + Grafana stack** — docker-compose, datasources
  and dashboards provisioned from the repo
- [ ] **Deep inspection** — threads, FD tables, socket correlation
- [ ] **JSON detail API** — `/api/processes`, `/api/process/{pid}`,
  Grafana Infinity drill-down dashboards
- [ ] **Host context** — CPU/memory/disk capacity metrics beside the
  per-process numbers
- [ ] **Alerting** — Alertmanager, recording + alerting rules, a
  runbook per alert
- [ ] **Hardening** — health endpoints, graceful shutdown, systemd
  unit, multi-day soak
- [ ] **MCP server** — typed, read-only tools (process listing and
  inspection, PromQL queries, active alerts) over the existing API
  for MCP-compatible clients

## Design rules

- Parsers are stdlib-only and pure: bytes in, typed structs out. File
  I/O lives in a thin outer layer; `internal/procfs` never logs or
  prints.
- Units live in names (`VmRSSKiB`, `UTimeTicks`, `RSSPages`);
  conversion happens only at display and export boundaries.
- Three error classes, three behaviors: process vanished (normal —
  skip or degrade), permission denied (degrade that item, continue),
  malformed content (fail loudly, naming the file).
- Fixtures over mocks: parsers are tested against committed captures
  of real kernel output.
- `/proc/<pid>/environ` is never read. HTTP surfaces bind localhost
  unless a decision record says otherwise.

## Documentation

- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — target architecture
  and current state
- [docs/runbook.md](docs/runbook.md) — what runs today, exactly
- [docs/decisions.md](docs/decisions.md) — decision records (ADRs)
- [docs/errors.md](docs/errors.md) — investigated surprises
- [AGENTS.md](AGENTS.md) / [CLAUDE.md](CLAUDE.md) — machine-readable
  contributor guides
