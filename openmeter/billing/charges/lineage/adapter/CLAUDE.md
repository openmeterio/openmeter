# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing lineage.Adapter for credit realization lineage and segment persistence. Pure persistence layer: all reads, bulk creates, row-locking FOR UPDATE queries, and close/create segment mutations — no business logic.

## Patterns

**TransactingRepo on every method body** — Every public method wraps its Ent calls in entutils.TransactingRepo or entutils.TransactingRepoWithNoValue so operations rebind to any transaction already in ctx rather than using the raw *entdb.Client field. (`return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error { _, err := tx.db.CreditRealizationLineage.CreateBulk(...).Save(ctx); return err })`)
**Tx/WithTx/Self triad** — adapter implements Tx(ctx) (context.Context, transaction.Driver, error), WithTx(ctx, *TxDriver) *adapter, and Self() *adapter — required by entutils.TransactingRepo to rebind to in-progress transactions. (`func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { txDB := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txDB.Client()} }`)
**GetDriverFromContext guard on lock methods** — LockCorrectionLineages and LockAdvanceLineagesForBackfill call entutils.GetDriverFromContext and return an error immediately if no active transaction is found — FOR UPDATE locks are only meaningful inside a transaction. (`if _, err := entutils.GetDriverFromContext(ctx); err != nil { return nil, fmt.Errorf("lock correction lineages must be called in a transaction: %w", err) }`)
**Package-level standalone helper** — LoadActiveSegmentsByRealizationID is exposed as a package-level function (not only a method) so other sub-packages can reuse it without constructing a full adapter through DI; internally it wraps a temporary &adapter{db: db} in TransactingRepo. (`func LoadActiveSegmentsByRealizationID(ctx context.Context, db *entdb.Client, ...) (...) { repo := &adapter{db: db}; return entutils.TransactingRepo(ctx, repo, ...) }`)
**Config/Validate/New constructor** — New(Config) validates required fields (Client not nil) via Config.Validate() and returns (lineage.Adapter, error) — consistent adapter constructor shape across the billing domain. (`func New(config Config) (lineage.Adapter, error) { if err := config.Validate(); err != nil { return nil, err }; return &adapter{db: config.Client}, nil }`)
**Private mapLineage/mapSegment conversion helpers** — Unexported pure functions translate Ent rows to domain types, keeping conversion centralized and isolating Ent internals from callers. (`func mapSegment(segment *entdb.CreditRealizationLineageSegment) lineage.Segment { return lineage.Segment{ID: segment.ID, LineageID: segment.LineageID, ...} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Defines Config, New constructor, and the adapter struct with Tx/WithTx/Self — the minimal scaffold required by entutils.TransactingRepo. No business logic. | Never use a.db directly inside a method body — always use tx.db inside the TransactingRepo closure. Struct field db is overridden by WithTx when a caller tx is active. |
| `lineage.go` | Implements all lineage.Adapter methods: bulk create lineages/segments, load active segments by realization ID or customer, close/create individual segments, and FOR UPDATE locking queries for correction and backfill paths. | LockCorrectionLineages and LockAdvanceLineagesForBackfill call ForUpdate() — always verify GetDriverFromContext before issuing these. The standalone LoadActiveSegmentsByRealizationID must also use TransactingRepo, not raw db. Segments have no UpdatedAt — close-and-create is the only mutation path. |

## Anti-Patterns

- Calling tx.db.X directly in a method body without entutils.TransactingRepo/TransactingRepoWithNoValue — falls off the caller's transaction and produces partial writes
- Removing the GetDriverFromContext guard from lock methods — makes FOR UPDATE semantics unreliable outside explicit transactions
- Adding business or computation logic to the adapter — persistence only; service logic belongs in lineage/service
- Returning *adapter publicly instead of the lineage.Adapter interface — leaks implementation and breaks testability
- Constructing adapter{db: db} inline outside New() or WithTx() — bypasses validation and breaks the TxCreator contract

## Decisions

- **Every method wraps Ent calls in TransactingRepo even when no caller tx is present** — Charge advancement mixes multiple reads and writes; a helper that falls off the transaction produces partial writes. TransactingRepo rebinds to the existing tx in ctx or starts a new one automatically.
- **Lock methods assert an active transaction via GetDriverFromContext before issuing FOR UPDATE** — FOR UPDATE row locks are only useful inside a transaction. An explicit guard surfaces programmer errors early rather than silently issuing no-op locks.
- **Package-level LoadActiveSegmentsByRealizationID alongside the method** — Other sub-packages in billing/charges need lineage segments without constructing a full adapter through DI; the standalone function accepts a raw *entdb.Client and internally wraps it in TransactingRepo for safety.

## Example: Adding a new adapter method that reads and writes lineage segments atomically

```
import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func (a *adapter) MyNewOperation(ctx context.Context, input lineage.MyInput) error {
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		// Always use tx.db — never a.db — inside this closure
		if _, err := tx.db.CreditRealizationLineage.Query()...; err != nil {
			return fmt.Errorf("my new operation: %w", err)
		}
		return nil
// ...
```

<!-- archie:ai-end -->
