# Enforcement: concurrency (4 rules)

Topic file. Loaded on demand when an agent works on something in the `concurrency` area. The pre-edit hook reads `.archie/rules.json` directly — this file is for browsing/context only.

## Decision Violations (block)

### `dec-lock-001` — Serialize per-entity multi-row operations with a lockr advisory lock keyed on a globally-unique id inside transaction.Run

*source: `deep_scan`*

**Why:** Subscription mutations, billing mutations, and charge advances each touch many rows/tables for one customer and must be serialized per logical entity without serializing the whole table. lockr.NewKey(scopes...) xxh3-hashes the scope to a 64-bit int and Locker.LockForTX issues SELECT pg_advisory_xact_lock($1) on the current Ent tx, auto-releasing on commit/rollback. getTxClient hard-asserts the caller is inside a real Postgres transaction (transaction_timestamp() != statement_timestamp()) and errors otherwise (pkg/framework/lockr/locker.go:134). The key id component must be PK-unique (subscription.GetCustomerLock keys on customer id).

**Example:**

```
err := transaction.Run(ctx, svc.adapter, func(ctx context.Context) error {
    if err := svc.locker.LockForTX(ctx, subscription.GetCustomerLock(customerID)); err != nil { return err }
    return svc.mutate(ctx, ...)
})
```

## Pitfalls (block)

### `pf-context-001` — Never introduce context.Background() or context.TODO() to sidestep missing context propagation in app code

*source: `deep_scan`*

**Why:** Context-ambient transactions carry the active tx on context.Context, rebound by entutils.TransactingRepo and not visible in method signatures. Introducing context.Background()/context.TODO() in application code detaches the call from the caller's transaction, cancellation, deadlines, and tracing — and a write that loses the ctx tx silently commits outside the caller's transaction. AGENTS.md forbids this; either propagate the caller's ctx through the full path, or drop the unused context.Context parameter.

**Path glob:** `openmeter/**`, `pkg/**`, `app/**`, `api/**`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "context\\.Background\\(\\)",
      "context\\.TODO\\(\\)"
    ],
    "must_not_match": [
      "_test\\.go",
      "main\\.go"
    ]
  }
]
```

</details>

## Tradeoff Signals (warn)

### `tr-lock-002` — Never key a lockr advisory lock on a non-unique column such as customer.key

*source: `deep_scan`*

**Why:** Advisory locks must key on a globally-unique id; misuse is only caught at runtime. customer.key is only unique under namespace + deleted_at IS NULL (openmeter/ent/schema/customer.go:58-62), so a key-based lock could serialize unrelated customers across namespaces or collide a live row with a soft-deleted one. Key on the PK-unique id instead.

**Path glob:** `openmeter/**`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "lockr\\.NewKey\\([^)]*\\.Key",
      "lockr\\.NewKey\\([^)]*\"key\""
    ]
  }
]
```

</details>

### `tr-lock-003` — Do not use sync.Mutex or session-scoped pg_advisory_lock for cross-replica serialization

*source: `deep_scan`*

**Why:** The worker binaries are horizontally scaled, so an in-process sync.Mutex would not serialize across replicas. Session-scoped pg_advisory_lock (not pg_advisory_xact_lock) ties lock lifetime to the connection, not the transaction, re-introducing lock-then-die orphan windows that the transaction-scoped lockr design eliminates.

**Path glob:** `openmeter/**`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "pg_advisory_lock\\("
    ]
  }
]
```

</details>
