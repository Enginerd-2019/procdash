# Errors — investigated surprises

Failures that actually happened, cost real investigation time, and
were not anticipated — recorded so the same investigation never runs
twice. Expected degradations (permission-denied on another user's
`io`, vanished processes) are normal behavior and do not belong here;
see the runbook.

**Format** (newest first, numbered sequentially):

```
## E-NNN — one-line symptom (YYYY-MM-DD)

**Symptom:** what was observed, verbatim where possible.
**Root cause:** the actual mechanism.
**Fix:** what resolved it (commit reference if code changed).
**Prevention:** what now guards against recurrence — a fixture, a
test, a runbook line, or "awareness only."
```

---

*No entries yet.*
