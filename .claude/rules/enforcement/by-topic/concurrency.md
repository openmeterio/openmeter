# Enforcement: concurrency (1 rule)

Topic file. Loaded on demand when an agent works on something in the `concurrency` area. The pre-edit hook reads `.archie/rules.json` directly — this file is for browsing/context only.

## Tradeoff Signals (warn)

### `entitlement-001` — Entitlement operations that modify multiple entitlement rows for the same customer must acquire a pg_advisory_lock per customer via lockr.Locker before beginning mutations.

*source: `deep_scan`*

**Why:** The openmeter/entitlement component description states: 'Acquires pg_advisory_lock per customer before operations modifying multiple entitlement rows.' Without this lock, concurrent balance recalculation in balance-worker and entitlement mutations from the API server can race, producing split-brain grant burn-down snapshots that are invisible at the query level.

**Example:**

```
// Correct: acquire lock before multi-row entitlement mutation
return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*Entitlement, error) {
    if err := locker.LockForTX(ctx, lockr.NewKey("entitlement", customerID)); err != nil {
        return nil, err
    }
    // ... mutations ...
})
```
