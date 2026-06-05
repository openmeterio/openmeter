# transaction

<!-- archie:ai-start -->

> Defines the transaction abstraction (Driver/Creator interfaces) and the Run/RunWithNoValue helpers that execute a callback inside a transaction, plus context-key plumbing to store and retrieve the active Driver. This is the foundation that entutils.TransactingRepo and every adapter layer build on for nested, savepoint-based transactions.

## Patterns

**Run/RunWithNoValue entrypoints** — All transactional work goes through Run[R] (returns a value) or RunWithNoValue (delegates to Run with a nil any return); never call Driver.Commit/Rollback directly from business code. (`transaction.RunWithNoValue(ctx, a.creator, func(ctx context.Context) error { ... })`)
**Context-carried Driver reuse** — getTx first checks GetDriverFromContext; if a Driver is already present it is reused (joins the outer tx) instead of opening a new one, enabling nested transactions across services. (`if tx, err := GetDriverFromContext(ctx); err == nil { return ctx, tx, nil }`)
**Savepoint per Run invocation** — manage() calls tx.SavePoint() before the callback so nested Run calls roll back to their own savepoint rather than aborting the whole outer transaction. (`err := tx.SavePoint(); result, err := cb(ctx, tx)`)
**Single typed context key** — Driver is stored under the unexported omTransactionContextKey constant; SetDriverOnContext returns DriverConflictError if one already exists rather than overwriting. (`context.WithValue(ctx, contextKey, tx)`)
**Sentinel error types for tx state** — DriverNotFoundError and DriverConflictError are concrete types checked via type assertion (not errors.Is); getTx tolerates DriverNotFoundError silently and Run tolerates DriverConflictError (nested case). (`if _, ok := err.(*DriverConflictError); !ok && err != nil { ... }`)
**Panic rolls back then re-panics** — manage()'s deferred recover rolls the tx back for all downstream WithTx clients and re-panics with a message including debug.Stack() — it does not convert panic to error. (`_ = tx.Rollback(); panic(pMsg)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `transaction.go` | Driver/Creator interfaces and Run/RunWithNoValue/getTx/manage; the transaction execution engine. | On callback error, Rollback errors are joined via errors.Join and returned; a failed Commit also triggers Rollback. Driver must implement Commit/Rollback/SavePoint — SavePoint is what makes nesting safe. |
| `context.go` | Get/SetDriverOnContext plus DriverNotFoundError and DriverConflictError sentinel types keyed by omTransactionContextKey. | SetDriverOnContext refuses to replace an existing Driver (returns DriverConflictError); Run treats that conflict as the expected nested case and proceeds. Errors are matched by type assertion, so don't wrap them in ways that defeat the cast. |

## Anti-Patterns

- Calling Driver.Commit()/Rollback() directly from a service or adapter instead of going through Run.
- Opening a fresh transaction when ctx already carries a Driver — bypasses nested-tx reuse and savepoints.
- Overwriting the context Driver under a different key, breaking GetDriverFromContext reuse.
- Recovering panics inside the callback to avoid rollback — manage() relies on the panic propagating to roll back.
- Treating DriverNotFoundError/DriverConflictError with errors.Is instead of the concrete type assertion the package uses.

## Decisions

- **Reuse a context-stored Driver and use savepoints** — Lets independently-written services compose into one outer transaction; each Run gets its own savepoint so a nested failure rolls back only its scope unless the error propagates.
- **Panic path rolls back and re-panics rather than returning an error** — Guarantees downstream WithTx clients see a rolled-back tx while preserving the original panic + stack for higher-level recovery/observability.
- **Concrete sentinel error types over errors.New** — getTx and Run distinguish 'no tx yet' (open one) from 'tx already present' (reuse) via type assertion, which sentinel strings could not express cleanly.

## Example: Run repository work inside a transaction

```
import "github.com/openmeterio/openmeter/pkg/framework/transaction"

func (a *adapter) Create(ctx context.Context, in Input) (*Out, error) {
    return transaction.Run(ctx, a.creator, func(ctx context.Context) (*Out, error) {
        // adapter reads the active Driver from ctx (via entutils.TransactingRepo)
        return a.doCreate(ctx, in)
    })
}
```

<!-- archie:ai-end -->
