# Decision records

Choices with lasting consequences, recorded so they can be cited
(ADR-NNN) and revisited deliberately instead of re-argued. When a
decision is overturned, its status changes to "superseded by
ADR-NNN" — records are never deleted.

## ADR-001 — Hand-rolled, stdlib-only /proc parsers

**Date:** 2026-08-14 · **Status:** accepted
**Context:** Mature libraries (`gopsutil`, `prometheus/procfs`) parse
`/proc` already.
**Decision:** Parse `/proc` with the standard library only. Mature
libraries serve as comparison references, never as dependencies. The
first external module (`prometheus/client_golang`) arrives only with
the exporter.
**Consequences:** Every exported number is auditable end-to-end;
structs are shaped exactly for the collector with no adaptation
layer; the foundation stays dependency-free. Cost: this project owns
every format edge case — mitigated by the fixture harness (ADR-005).

## ADR-002 — Two data planes, split by data shape

**Date:** 2026-08-14 · **Status:** accepted
**Context:** Grafana can be fed by Prometheus, by JSON (Infinity
datasource), or both. Process data is part numeric time series, part
unbounded-cardinality detail (FD tables, connection tuples).
**Decision:** Numeric time-series data goes through Prometheus
(history, `rate()`, alerting). Point-in-time rich detail goes through
a JSON API rendered via Infinity. One binary serves both. Detail
never enters metric labels.
**Consequences:** Each data shape gets the transport it is good at;
metric cardinality stays bounded by design. Cost: two panel types and
a drill-down seam to maintain. This is the project's core
architecture decision.

## ADR-003 — Exporter native on the host; stack in containers

**Date:** 2026-08-14 · **Status:** accepted
**Context:** procdash needs the real host `/proc`; Prometheus and
Grafana are stateless-config infrastructure.
**Decision:** procdash runs natively on the host (eventually under
systemd). Prometheus, Grafana, and Alertmanager run under
docker-compose in `deploy/`, scraping the host exporter.
**Consequences:** No `/proc`-into-container mounting contortions;
clean separation of the thing that measures from the things that
store and render. Cost: two lifecycles to manage.

## ADR-004 — Observability configuration as code

**Date:** 2026-08-14 · **Status:** accepted
**Context:** Dashboards and datasources click-built inside a Grafana
volume are unreproducible and unreviewable.
**Decision:** Prometheus config, Grafana datasources, dashboard JSON,
and alert rules live in the repo and provision automatically.
`docker compose up` on a clean machine must reproduce the entire
stack.
**Consequences:** Reviewable diffs; reproducibility. Cost: the
export-dashboard-to-JSON loop after UI edits — the standard industry
workflow.

## ADR-005 — Injectable procfs root as the universal test seam

**Date:** 2026-08-14 · **Status:** accepted
**Context:** Parsers need regression tests for states that cannot be
produced live on demand (hostile process names, zombies, vanished
processes, kernel format drift).
**Decision:** All procfs access goes through an `FS{root}` value
(default `/proc`). Tests run against committed fixture trees under
`internal/procfs/testdata/` — live-captured kernel output preferred;
hand-crafted fixtures labeled as such.
**Consequences:** Every parser bug becomes a permanent, replayable
fixture; CI needs no live kernel weirdness; every later procfs
addition inherits the harness. Cost: none meaningful.

## ADR-006 — Units in names; conversion at boundaries only

**Date:** 2026-08-14 · **Status:** accepted
**Context:** The same quantity appears as KiB (`status`), pages
(`stat`), bytes (`io`), and ticks (`stat`) — a silent-×1024 bug
factory.
**Decision:** Struct fields record what the kernel said, in the
kernel's units, with the unit in the field name (`VmRSSKiB`,
`UTimeTicks`, `RSSPages`). Conversions live in methods and at
display/export boundaries. `userHZ = 100` is a documented constant;
page size comes from `os.Getpagesize()`.
**Consequences:** A wrong unit becomes a visible name mismatch at the
call site instead of a silent factor-of-1024 error. Prometheus
base-unit conversion happens in the collector, keeping parsers
faithful witnesses.

## ADR-007 — Error taxonomy: gone / not-permitted / malformed

**Date:** 2026-08-14 · **Status:** accepted
**Context:** An inspector's world is partially visible by design:
processes vanish mid-read; other users' files deny access.
**Decision:** Three outcomes, three behaviors: vanished →
`procfs.ErrGone` sentinel (normal; callers skip or degrade); denied →
wrapped `fs.ErrPermission` (degrade the affected output, continue);
malformed → loud descriptive parse error (a bug or kernel drift).
The library never logs or prints; presentation belongs to callers.
**Consequences:** Every consumer inherits one policy instead of
improvising per call site. Folding denied into gone would make the
tool report "process gone" for processes that are demonstrably alive
whenever it lacks privileges — a lie that surfaces only for
unprivileged users and would later produce false "process absent"
alerts.
