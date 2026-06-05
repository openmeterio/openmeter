# Enforcement: data-access (4 rules)

Topic file. Loaded on demand when an agent works on something in the `data-access` area. The pre-edit hook reads `.archie/rules.json` directly — this file is for browsing/context only.

## Decision Violations (block)

### `dec-tx-001` — Ent adapter methods must wrap their body in entutils.TransactingRepo so they join the caller's transaction

*source: `deep_scan`*

**Why:** This is the seam that lets one domain's service participate in another domain's transaction over the single shared Ent client. Without it, subscription→billing→charges→ledger composition could not be atomic. Every domain adapter holds a *entdb.Client and implements entutils.TxCreator (Tx() hijacks an Ent tx onto the context) and TxUser[T] (WithTx rebinds the adapter to the tx client; Self() returns the non-tx instance); TransactingRepo rebinds to the *TxDriver already on the context if one exists and otherwise falls back to repo.Self() (pkg/framework/entutils/transaction.go:199).

**Example:**

```
func (a *adapter) GetCustomer(ctx context.Context, id models.NamespacedID) (*customer.Customer, error) {
    return entutils.TransactingRepo(ctx, a, func(ctx context.Context, rep *adapter) (*customer.Customer, error) {
        row, err := rep.db.Customer.Query().Where(customerdb.Namespace(id.Namespace), customerdb.ID(id.ID)).Only(ctx)
        if err != nil { return nil, err }
        return mapCustomerFromDB(row), nil
    })
}
```

**Path glob:** `openmeter/**/adapter/**`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "\\.db\\.\\w+\\.(Create|Update|Delete|Query)\\("
    ],
    "must_not_match": [
      "entutils\\.TransactingRepo",
      "entutils\\.TransactingRepoWithNoValue"
    ]
  }
]
```

</details>

## Pattern Divergence (inform)

### `dec-tx-002` — Charges adapter helpers taking a raw *entdb.Client must still wrap their body with entutils.TransactingRepo

*source: `deep_scan`*

**Why:** In openmeter/billing/charges/.../adapter, keep Ent access transaction-aware: even helpers that accept a raw *entdb.Client argument must wrap their body with entutils.TransactingRepo / TransactingRepoWithNoValue so they rebind to the tx already in ctx. Passing a non-tx client silently writes outside the caller's transaction, breaking the cross-domain atomicity the architecture depends on.

**Path glob:** `openmeter/billing/charges/**/adapter/**`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "func \\w+\\([^)]*\\*entdb\\.Client[^)]*\\)"
    ],
    "must_not_match": [
      "entutils\\.TransactingRepo"
    ]
  }
]
```

</details>

### `data-clickhouse-store-001` — Store usage events only in ClickHouse; never persist or join RawEvent against Postgres

*source: `deep_scan`*

**Why:** A separate ClickHouse store and Kafka pipeline carry usage events: per-namespace om_<ns>_events MergeTree tables, written by the sink-worker. Storing usage events in Postgres, joining RawEvent against Postgres tables, or removing the sink-worker collapses the read-heavy append-only data plane into the OLTP control plane it was deliberately split from.

### `data-redis-001` — Use Redis only for dedupe and async-query progress, with TTL keys, not as a source of truth

*source: `deep_scan`*

**Why:** Redis is a cache/coordination store: ingest event deduplication via SET NX keys with TTL (openmeter/dedupe/redisdedupe) and async query progress under progress:<ns>:<id> keys with TTL (openmeter/progressmanager/adapter). It is role=cache, not a primary store; durable control-plane state belongs in Postgres.
