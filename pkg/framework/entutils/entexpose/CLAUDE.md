# entexpose

<!-- archie:ai-start -->

> An Ent codegen extension that exposes generated client internals (driver/config) and adds HijackTx / NewTxClientFromRawConfig so multiple db.Client and db.Tx instances over the same connection can share a single transaction. Must be included in every Ent codegen package that participates in shared transactions.

## Patterns

**entc.Extension wrapper** — Identical registration shape to the other entutils extensions: Extension embeds entc.DefaultExtension, Templates() registers the embedded expose.tpl under name 'entexpose', New() returns *Extension. (`gen.MustParse(gen.NewTemplate("entexpose").Parse(tmplfile))`)
**Internal exposure for shared tx** — The template emits Client.GetConfig() returning *entutils.RawEntConfig and an ExposedTxDriver wrapping *txDriver that implements entutils.Transactable (Rollback/Commit/SavePoint/RollbackTo/Release). (`var _ entutils.Transactable = (*ExposedTxDriver)(nil)`)
**HijackTx / rehydrate** — Client.HijackTx begins a tx and returns a RawEntConfig + ExposedTxDriver; NewTxClientFromRawConfig rebuilds a *Tx (with all node clients) from that raw config so a transaction can be shared across Ent codegen packages. (`func (c *Client) HijackTx(ctx, opts) (context.Context, *entutils.RawEntConfig, *ExposedTxDriver, error)`)
**Savepoint via raw SQL** — SavePoint/RollbackTo/Release execute literal SAVEPOINT / ROLLBACK TO / RELEASE SAVEPOINT statements through the txDriver's ExecContext. (`d.Driver.ExecContext(context.Background(), "SAVEPOINT " + name)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `entexpose.go` | Registers the entexpose template as an entc.Extension via embedded expose.tpl | Template define name 'expose' vs registration name 'entexpose' — registration uses gen.NewTemplate name, define block is named 'expose' |
| `expose.tpl` | Generates GetConfig, ExposedTxDriver, HijackTx, NewTxClientFromRawConfig into each participating Ent package | Known limitation noted in template TODO: Tx.onRollback/onCommit hooks and intersectors are ignored under shared transactions; HijackTx errors if already inside a txDriver |

## Anti-Patterns

- Using shared transactions and relying on Ent Tx onRollback/onCommit hooks (they are ignored by this template)
- Forgetting to include this extension in a new Ent codegen package that must join shared transactions
- Editing the generated GetConfig/HijackTx output instead of expose.tpl
- Calling HijackTx on a client already running inside a transaction (returns an error)

## Decisions

- **Expose driver/config internals via template-generated methods on the generated Client** — Cross-package shared transactions require reaching the unexported config/driver that Ent does not export by default
- **Model savepoints with raw SQL through ExposedTxDriver** — Gives entutils.Transactable nested-transaction semantics over a hijacked tx without Ent's native support

<!-- archie:ai-end -->
