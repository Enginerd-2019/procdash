# Fixtures

Captured `/proc` files the parser tests run against. The suite needs
no root and reads nothing live — every case is pinned to one of these
files. Each directory mirrors a `/proc/<pid>/` tree so the injectable
`FS` root can point at it.

| Path | File | Origin |
|---|---|---|
| `basic/600314/status` | status | Live-captured, unedited (a real `sh`). |
| `basic/1726633/stat` | stat | Live-captured, unedited (a real `sh`). |
| `kthread/2/status` | status | Live-captured kernel thread — no `Vm*` block, exercises `HasVM == false`. |
| `hostile-comm/1729364/stat` | stat | Live-captured. The process renamed itself to a value containing a space and parentheses, so a naive whitespace split mis-indexes every later field. |
| `truncated/1/stat` | stat | **Hand-crafted** — a stat line cut off mid-fields. No kernel emits this; it exists to prove a short read produces a loud parse error, not a panic. |

Live captures are `cp` of a real `/proc/<pid>/` file and are never
edited by hand. The one hand-crafted file is labelled above; any
future hand-crafted fixture is added to this table the same way.
