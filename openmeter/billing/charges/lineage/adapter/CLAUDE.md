# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing lineage.Adapter for credit realization lineage and segment persistence. Provides all DB reads, writes, and row-locking for the lineage domain, enabling correct multi-step credit realization tracking within billing charges.

## Patterns

**TransactingRepo wrapping all DB operations** — Every adapter method wraps its Ent queries in entutils.TransactingRepo or entutils.TransactingRepoWithNoValue so the operation rebinds to any transaction already carried in ctx, rather than using the raw *entdb.Client directly. (`return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error { _, err := tx.db.CreditRealizationLineage.CreateBulk(...).Save(ctx); return err })`)
**Tx/WithTx/Self triad for TxCreator compatibility** — adapter implements Tx(ctx) (context.Context, transaction.Driver, error), WithTx(ctx, *TxDriver) *adapter, and Self() *adapter — required by entutils.TransactingRepo to rebind the client to an in-progress transaction. (`func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { txDB := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txDB.Client()} }`)
**Explicit tx-required guard on lock operations** — Methods that issue FOR UPDATE SQL (LockCorrectionLineages, LockAdvanceLineagesForBackfill) call entutils.GetDriverFromContext and return an error if no active transaction is found, enforcing that callers hold a transaction before acquiring row locks. (`if _, err := entutils.GetDriverFromContext(ctx); err != nil { return nil, fmt.Errorf("lock correction lineages must be called in a transaction: %w", err) }`)
**Standalone package-level helper for shared cross-package use** — LoadActiveSegmentsByRealizationID is exposed as a standalone package-level function (not only a method) so other sub-packages can reuse it without needing to construct a full adapter. (`func LoadActiveSegmentsByRealizationID(ctx context.Context, db *entdb.Client, namespace string, realizationIDs []string) (lineage.ActiveSegmentsByRealizationID, error) { repo := &adapter{db: db}; return entutils.TransactingRepo(ctx, repo, ...) }`)
**Config struct + Validate() constructor pattern** — New(Config) takes a Config struct, calls Validate() to guard required fields, and returns (lineage.Adapter, error) — consistent with the adapter constructor pattern across the billing domain. (`func New(config Config) (lineage.Adapter, error) { if err := config.Validate(); err != nil { return nil, err }; return &adapter{db: config.Client}, nil }`)
**Private map helpers for Ent-to-domain conversion** — mapLineage and mapSegment are unexported pure functions that translate Ent rows to domain types, keeping conversion logic centralized and testable without depending on Ent internals from outside the package. (`func mapSegment(segment *entdb.CreditRealizationLineageSegment) lineage.Segment { return lineage.Segment{ID: segment.ID, LineageID: segment.LineageID, ...} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Defines Config, New constructor, and the adapter struct with Tx/WithTx/Self — the minimal scaffold required by entutils.TransactingRepo. No business logic here. | Never accept a raw *entdb.Client in method bodies without wrapping with TransactingRepo; the struct field db is always overridden by WithTx when a tx is active. |
| `lineage.go` | Implements all lineage.Adapter methods: bulk create lineages/segments, load active segments, close/create individual segments, and locking queries for correction and backfill paths. | LockCorrectionLineages and LockAdvanceLineagesForBackfill call ForUpdate() — always verify GetDriverFromContext before issuing these or the lock has no guarantees. The standalone LoadActiveSegmentsByRealizationID must also use TransactingRepo, not raw db. |

## Anti-Patterns

- Calling tx.db.X directly inside a method body without entutils.TransactingRepo/TransactingRepoWithNoValue — falls off the caller's transaction
- Removing the GetDriverFromContext guard from lock methods — makes FOR UPDATE locks unreliable outside explicit transactions
- Adding business or computation logic to the adapter — this package is persistence-only; service logic belongs in lineage/service
- Returning *adapter publicly instead of the lineage.Adapter interface — leaks implementation and breaks testability
- Constructing adapter{db: db} inline outside of New() or WithTx() — bypasses validation and breaks the TxCreator contract

## Decisions

- **All DB operations wrapped in TransactingRepo even when no caller tx is present** — Charge advancement mixes multiple reads and writes; a helper that falls off the transaction causes partial writes under concurrency. TransactingRepo either rebinds to the existing tx in ctx or starts a new one automatically.
- **Lock methods assert an active transaction via GetDriverFromContext** — FOR UPDATE row locks are only useful inside a transaction. An explicit guard surfaces programmer errors early rather than silently issuing a no-op lock.
- **Package-level LoadActiveSegmentsByRealizationID alongside the method** — Other sub-packages in billing/charges need to load lineage segments without constructing a full adapter through DI; the standalone function accepts a raw *entdb.Client and internally wraps it in TransactingRepo for safety.

## Example: Adding a new adapter method that reads and writes lineage segments in one operation

```
import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func (a *adapter) MyNewOperation(ctx context.Context, input lineage.MyInput) error {
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		// use tx.db — never a.db — inside this closure
		if _, err := tx.db.CreditRealizationLineage.Query()...; err != nil {
			return fmt.Errorf("my new operation: %w", err)
		}
		return nil
// ...
```

<!-- archie:ai-end -->
