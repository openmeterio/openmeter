# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing currencies.Adapter for custom currencies and cost bases. All DB access is transaction-aware via entutils.TransactingRepo; the adapter struct implements the TxCreator+TxUser triad required by entutils.TransactingRepo to start or join caller-supplied transactions.

## Patterns

**TransactingRepo wrapping every method** — Every public adapter method body is wrapped with entutils.TransactingRepo(ctx, a, func(ctx, tx) ...) — never call a.db directly without this wrapper so the ctx-bound Ent transaction is honoured. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (currencies.Currency, error) { curr, err := tx.db.CustomCurrency.Create()... })`)
**Tx / WithTx / Self triad** — The adapter implements TxCreator: Tx() hijacks a new pg transaction via a.db.HijackTx, WithTx() rebinds the adapter to an existing TxDriver using entdb.NewTxClientFromRawConfig, Self() returns itself. All three are required for TransactingRepo. (`func (a *adapter) Tx(ctx context.Context) (context.Context, transaction.Driver, error) { txCtx, rawConfig, eDriver, err := a.db.HijackTx(ctx, &sql.TxOptions{ReadOnly: false}); return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil }`)
**Config struct + Validate + New constructor** — Adapter is constructed via New(Config) with Config.Validate() guard; callers never instantiate the adapter struct directly. (`func New(config Config) (currencies.Adapter, error) { if err := config.Validate(); err != nil { return nil, err }; return &adapter{db: config.Client}, nil }`)
**Compile-time interface assertion** — Declare var _ currencies.Adapter = (*adapter)(nil) in both adapter.go and currencies.go to catch interface drift at compile time. (`var _ currencies.Adapter = (*adapter)(nil)`)
**entdb.IsConstraintError → GenericConflictError** — On Ent constraint violations wrap with models.NewGenericConflictError so the HTTP layer maps it to 409 automatically. (`if entdb.IsConstraintError(err) { return currencies.Currency{}, models.NewGenericConflictError(fmt.Errorf("currency with code %s already exists", params.Code)) }`)
**mapXFromDB mapper functions** — DB row → domain type conversion is always in package-level mapXFromDB functions (mapCurrencyFromDB, mapCostBasisFromDB); never inline struct construction in query handlers. (`func mapCurrencyFromDB(c *entdb.CustomCurrency) currencies.Currency { return currencies.Currency{NamespacedID: models.NamespacedID{ID: c.ID, Namespace: c.Namespace}, Code: c.Code, ...} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Defines Config, New constructor, the adapter struct, and the Tx/WithTx/Self transaction plumbing — the skeleton every new adapter method depends on. | Do not add business logic here; it is strictly wiring. If Self() or WithTx() is removed or signature changes, TransactingRepo breaks silently. |
| `currencies.go` | Implements all currencies.Adapter methods (ListCustomCurrencies, CreateCurrency, CreateCostBasis, ListCostBases) plus the DB→domain mappers. | ListCustomCurrencies uses manual Offset/Limit pagination; ListCostBases uses the generated q.Paginate helper — keep both consistent when extending. Mapper functions must normalize time to UTC (c.EffectiveFrom.In(time.UTC)). |

## Anti-Patterns

- Calling a.db.X directly inside a method body without entutils.TransactingRepo — bypasses the ctx-bound Ent transaction and produces partial writes
- Adding service-layer validation or business logic inside the adapter — belongs in openmeter/currencies/service
- Constructing *adapter directly instead of calling New(Config)
- Returning raw Ent errors to callers — always wrap or map to models.Generic* errors
- Storing *entdb.Tx as a struct field instead of using the TxDriver/TransactingRepo pattern

## Decisions

- **Tx/WithTx/Self triad inlined on the adapter struct rather than a separate TxWrapper type** — entutils.TransactingRepo requires the repository itself to implement TxCreator; inlining the triad avoids an extra indirection layer and matches the pattern used across all other domain adapters.
- **Separate mapper functions (mapCurrencyFromDB, mapCostBasisFromDB) rather than methods on Ent types** — Keeps generated Ent types isolated from domain types; changes to Ent schema only require updating the mapper, not every call site.

## Example: Adding a new adapter method that writes to the DB

```
import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (a *adapter) UpdateCurrency(ctx context.Context, params currencies.UpdateCurrencyInput) (currencies.Currency, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (currencies.Currency, error) {
		curr, err := tx.db.CustomCurrency.UpdateOneID(params.ID).
			SetName(params.Name).
			Save(ctx)
// ...
```

<!-- archie:ai-end -->
