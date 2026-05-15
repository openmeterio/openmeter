# tx

<!-- archie:ai-start -->

> Thin bridge that adapts the Ent ORM transaction API to the pkg/framework/transaction.Creator interface, allowing domain adapters and services to start Postgres transactions without importing ent internals directly.

## Patterns

**Implement transaction.Creator via db.HijackTx** — NewCreator(db *db.Client) returns a transaction.Creator whose Tx() method calls db.HijackTx to obtain a raw sql.Tx config, then wraps it in entutils.NewTxDriver. This is the only correct way to create a transaction.Driver that openmeter/pkg/framework/entutils.TransactingRepo can read from context. (`func (t *txCreator) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	txCtx, rawConfig, eDriver, err := t.db.HijackTx(ctx, &sql.TxOptions{ReadOnly: false})
	if err != nil { return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err) }
	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `enttx.go` | Sole file: exports NewCreator(db *db.Client) transaction.Creator. Wired in app/common/database.go and injected wherever a TxCreator is required. | Do not change sql.TxOptions to ReadOnly: true — billing adapters write inside transactions and will panic. Do not bypass this and construct transactions manually; downstream TransactingRepo depends on the ctx driver key set by HijackTx. |

## Anti-Patterns

- Constructing ent transactions outside this package (e.g. db.Tx(ctx) directly in an adapter) — bypasses the transaction.Driver protocol and makes the tx invisible to entutils.TransactingRepo.
- Adding business logic to this package — it is purely a bridge; keep it to one function.

## Decisions

- **Separate package rather than method on db.Client** — Keeps the ent-generated openmeter/ent/db package free of framework dependencies; domain adapters depend on the transaction.Creator interface without importing ent internals.

<!-- archie:ai-end -->
