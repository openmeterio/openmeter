# tx

<!-- archie:ai-start -->

> Thin bridge package (package enttx) exposing NewCreator(db *db.Client) transaction.Creator, the single adapter from Ent's *db.Client into the framework transaction.Driver protocol. Keeps the generated ent/db package free of framework dependencies.

## Patterns

**NewCreator is the only export** — Construct a transaction.Creator with NewCreator(db); wire it in app/common/database.go and inject wherever a TxCreator is needed. The package is a single function with no business logic. (`creator := enttx.NewCreator(dbClient)`)
**HijackTx + NewTxDriver in Tx()** — Tx() opens a writable transaction via db.HijackTx then wraps the result in entutils.NewTxDriver so downstream TransactingRepo can read the ctx-bound driver. (`txCtx, rawConfig, eDriver, err := t.db.HijackTx(ctx, &sql.TxOptions{ReadOnly: false}); return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `enttx.go` | Sole file: exports NewCreator(db *db.Client) transaction.Creator. | Do not change sql.TxOptions to ReadOnly:true — billing adapters write inside transactions and will panic. Do not bypass and build transactions manually; downstream TransactingRepo depends on the ctx driver key set by HijackTx. |

## Anti-Patterns

- Constructing ent transactions outside this package (e.g. db.Tx(ctx) directly in an adapter) — bypasses the transaction.Driver protocol and makes the tx invisible to entutils.TransactingRepo.
- Adding business logic here — it is purely a bridge; keep it to one function.

## Decisions

- **Separate package rather than a method on db.Client** — Keeps the ent-generated openmeter/ent/db package free of framework dependencies; domain adapters depend on the transaction.Creator interface without importing ent internals.

<!-- archie:ai-end -->
