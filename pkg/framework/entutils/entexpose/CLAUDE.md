# entexpose

<!-- archie:ai-start -->

> Ent code-generation extension that injects GetConfig(), HijackTx(), and NewTxClientFromRawConfig() into each generated db package, enabling the shared cross-client transaction mechanism used by entutils.TransactingRepo — the foundation of the ctx-propagated transaction pattern across all domain adapters.

## Patterns

**Ent Extension + embedded template** — entexpose.go implements entc.Extension with Templates() returning a gen.MustParse'd template from expose.tpl. Must be registered in openmeter/ent/entc.go. (`func New() *Extension { return &Extension{} }`)
**HijackTx for cross-client transaction sharing** — HijackTx(ctx, opts) starts a SQL transaction on the client's driver and returns (ctx, *RawEntConfig, *ExposedTxDriver, error). The RawEntConfig is passed to NewTxClientFromRawConfig in a second client to bind it to the same transaction. (`ctx, cfg, txDriver, err := client.HijackTx(ctx, &sql.TxOptions{})`)
**ExposedTxDriver implements Transactable with savepoints** — ExposedTxDriver exposes Rollback, Commit, SavePoint, RollbackTo, Release — all SAVEPOINT SQL issued via ExecContext on the txDriver. This lets entutils.TransactingRepo nest transactions. (`var _ entutils.Transactable = (*ExposedTxDriver)(nil)`)
**NewTxClientFromRawConfig reconstructs a Tx with empty hooks/inters** — Deliberately creates a Tx with empty hooks and inters structs — meaning Ent on-rollback/on-commit hooks are ignored for shared transactions (per the template TODO). (`config := config{driver: cfg.Driver, debug: cfg.Debug, log: cfg.Log, hooks: &hooks{}, inters: &inters{}}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `expose.tpl` | Core template generating GetConfig, HijackTx, NewTxClientFromRawConfig, and ExposedTxDriver into the db package. Changes affect ALL generated Ent packages. | Tx.onRollback and Tx.onCommit hooks are silently dropped in shared transactions — do not rely on Ent commit/rollback hooks in shared-tx flows. |
| `entexpose.go` | Extension registration — consumed by entc.Generate in openmeter/ent/entc.go. | Must stay in sync with pkg/framework/entutils types (RawEntConfig, Transactable) — the template references them by name. |

## Anti-Patterns

- Starting a transaction with HijackTx on a client that already has a txDriver — the template returns an error for nested HijackTx
- Relying on Ent commit/rollback hooks in shared-transaction paths — NewTxClientFromRawConfig initialises empty hooks
- Editing expose.tpl without regenerating openmeter/ent/db/ via make generate

## Decisions

- **Expose raw txDriver internals via code generation rather than a public Ent API** — Ent has no first-class API for sharing a transaction across multiple Client instances; hijacking the internal txDriver is the only way without forking Ent.

<!-- archie:ai-end -->
