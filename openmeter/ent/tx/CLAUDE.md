# tx

<!-- archie:ai-start -->

> Single-file adapter that bridges the generated Ent client to the generic transaction.Creator abstraction in pkg/framework/transaction, so domain repos can open transactions without importing the concrete ent.Client.

## Patterns

**Ent-backed transaction.Creator** — NewCreator(db *db.Client) returns a transaction.Creator whose Tx(ctx) hijacks an ent tx and wraps it with entutils.NewTxDriver, returning a context that carries the tx. (`func NewCreator(db *db.Client) transaction.Creator { return &txCreator{db: db} }`)
**HijackTx + NewTxDriver wrapping** — Tx() calls db.HijackTx(ctx, &sql.TxOptions{ReadOnly:false}) and returns entutils.NewTxDriver(eDriver, rawConfig) as the transaction.Driver; errors are wrapped with fmt.Errorf(...%w...). (`txCtx, rawConfig, eDriver, err := t.db.HijackTx(ctx, &sql.TxOptions{ReadOnly: false})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `enttx.go` | Defines unexported txCreator{db *db.Client} and exported NewCreator; the only public surface of package enttx. | Tx is hardcoded ReadOnly:false; the returned txCtx must be threaded down so repos rebind to the tx (entutils.TransactingRepo) — discarding it loses transaction propagation. |

## Anti-Patterns

- Importing openmeter/ent/db directly in domain repos instead of depending on transaction.Creator from this adapter.
- Swallowing the context returned by Tx() — downstream entutils.TransactingRepo relies on the tx carried in ctx.

## Decisions

- **Wrap ent transactions behind transaction.Creator** — Keeps the wide set of consumers (app/common, ledger/*, entitlement/metered, registry/builder, billing subscriptionsync) decoupled from the concrete ent client and uniform with pkg/framework/transaction semantics.

## Example: Wiring an ent client as a transaction.Creator

```
import (
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

var creator transaction.Creator = enttx.NewCreator(entClient) // entClient *db.Client
```

<!-- archie:ai-end -->
