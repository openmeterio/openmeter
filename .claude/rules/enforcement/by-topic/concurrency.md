# Enforcement: concurrency (2 rules)

Topic file. Loaded on demand when an agent works on something in the `concurrency` area. The pre-edit hook reads `.archie/rules.json` directly — this file is for browsing/context only.

## Pitfalls (block)

### `pf-008-ctxless-callback-background` — When an internal helper or third-party callback signature omits context.Context, refactor the signature to accept and thread the caller's ctx rather than substituting context.Background(); for fixed third-party callbacks, capture the request in the enclosing closure and use r.Context().

*source: `deep_scan`*

**Why:** Pitfall pf_0008: helper and callback functions whose signatures omit context.Context force callers to substitute context.Background(), severing cancellation, deadlines, OTel spans, and the ctx-bound Ent transaction driver. server.go:213 substitutes context.Background() in the v1 OapiRequestValidatorWithOptions ErrorHandler; appclient.go:240 calls UpdateAppStatus with context.Background() from providerError which takes no ctx. models.NewStatusProblem reads request-id from Chi middleware ctx, so a background ctx drops correlation from every v1 validation-error response.

**Example:**

```
// Wrong: providerError lacks ctx and falls back to Background
func (c *stripeAppClient) providerError(err error) error { c.svc.UpdateAppStatus(context.Background(), ...) }

// Right: thread ctx through the helper signature
func (c *stripeAppClient) providerError(ctx context.Context, err error) error { c.svc.UpdateAppStatus(ctx, ...) }
```

**Path glob:** `openmeter/app/stripe/**/*.go`, `openmeter/server/**/*.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "context\\.Background\\(\\)"
    ],
    "must_not_match": [
      "func main",
      "Shutdown"
    ]
  }
]
```

</details>

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
