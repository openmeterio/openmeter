# transaction

<!-- archie:ai-start -->

> Database-agnostic transaction abstraction: propagates a Driver (commit/rollback/savepoint) through context and provides Run/RunWithNoValue helpers that start-or-reuse a transaction, set savepoints for nested calls, and rollback on error or panic.

## Patterns

**Run[R] for transactional value-returning operations** — Call transaction.Run(ctx, creator, cb) to execute cb inside a transaction. If a Driver is already in ctx it is reused with a savepoint; otherwise creator.Tx starts a new one. (`result, err := transaction.Run(ctx, a.db, func(ctx context.Context) (*Entity, error) { return a.insertRow(ctx) })`)
**RunWithNoValue for error-only transactional operations** — Use transaction.RunWithNoValue(ctx, creator, cb) when the callback returns only error. Delegates to Run[interface{}] internally. (`err := transaction.RunWithNoValue(ctx, a.db, func(ctx context.Context) error { return a.deleteRow(ctx, id) })`)
**Driver propagation via context key** — SetDriverOnContext stores a Driver under a package-private key; GetDriverFromContext retrieves it. Run handles this automatically — adapters must not call SetDriverOnContext directly. (`tx, err := transaction.GetDriverFromContext(ctx) // read-only access; Run writes it`)
**Creator interface for adapter integration** — Any adapter that can start a transaction implements transaction.Creator via Tx(ctx) returning (ctx, Driver, error). Wire injects the concrete Creator into services. (`type Creator interface { Tx(ctx context.Context) (context.Context, Driver, error) }`)
**Savepoints for nested Run calls** — manage() calls tx.SavePoint() before every cb execution, enabling partial rollback within an outer transaction — required for multi-step billing mutations. (`// Automatic: every transaction.Run call creates a savepoint even when reusing an existing tx`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `context.go` | Typed context key, GetDriverFromContext, SetDriverOnContext, DriverNotFoundError, DriverConflictError. | SetDriverOnContext returns DriverConflictError (not nil) when a driver already exists — Run explicitly ignores this error type to reuse the existing tx; do not treat it as fatal outside Run. |
| `transaction.go` | Driver and Creator interfaces; Run, RunWithNoValue, getTx, manage. manage owns panic recovery (rollback + re-panic), savepoint creation, and commit/rollback on error. | manage() uses defer+recover to rollback and re-panic on any panic — never suppress this panic in adapter code. getTx silently reuses an existing Driver from ctx if present. |

## Anti-Patterns

- Calling creator.Tx() directly in adapter code instead of using transaction.Run — bypasses savepoint and rollback logic
- Storing a *entdb.Tx or raw DB client on adapter structs and using them directly — falls off the ctx-bound transaction
- Calling SetDriverOnContext manually outside of transaction.Run — Run already handles this; double-setting returns DriverConflictError and breaks nesting
- Using context.Background() inside a Run callback — loses the Driver already stored in ctx, breaking nested transaction reuse
- Ignoring DriverConflictError from SetDriverOnContext as if it were a hard error — it signals tx reuse and must not be treated as fatal

## Decisions

- **Driver interface with SavePoint/Commit/Rollback instead of exposing *sql.Tx** — Keeps the transaction package DB-agnostic; Ent, pgx, and any future driver can implement Driver without leaking ORM types into the framework layer.
- **Savepoints used for every nested Run call** — Allows partial rollback of nested operations while keeping the outer transaction alive, required for multi-step charge advancement and billing invoice mutations.
- **Panic recovery inside manage() rolls back then re-panics** — The DB transaction must be cleaned up even on unexpected panics, but the panic must propagate so upper-layer recovery (Chi middleware, test harness) still sees it.

## Example: Adapter method that must run inside a transaction, reusing an existing one from ctx if present

```
import (
	"context"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (a *adapter) CreateAndLink(ctx context.Context, in CreateInput) (*Entity, error) {
	return transaction.Run(ctx, a.db, func(ctx context.Context) (*Entity, error) {
		e, err := a.insertEntity(ctx, in)
		if err != nil {
			return nil, err
		}
		if err := a.insertLink(ctx, e.ID); err != nil {
			return nil, err // triggers Rollback via manage()
		}
		return e, nil
// ...
```

<!-- archie:ai-end -->
