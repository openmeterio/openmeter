# service

<!-- archie:ai-start -->

> Business-logic service implementing lineage.Service for credit realization lineage lifecycle: creating initial lineages, persisting correction segments, and backfilling advance segments. Orchestrates multi-step segment mutation sequences within transactions, delegating all persistence to lineage.Adapter.

## Patterns

**transaction.RunWithNoValue for all multi-step mutations** — Every mutating method wraps its full sequence in transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error { ... }) to ensure atomicity — the service holds an Adapter interface (not concrete *adapter) so it uses the transaction.Runner abstraction rather than entutils.TransactingRepo directly. (`return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error { ...; return s.adapter.CreateLineages(ctx, ...) })`)
**Input.Validate() before any business logic** — Every public method calls input.Validate() as its first operation and returns early on failure — before acquiring locks or performing DB reads. (`func (s *service) CreateInitialLineages(ctx context.Context, input lineage.CreateInitialLineagesInput) error { if err := input.Validate(); err != nil { return err }; ... }`)
**Config/Validate/New constructor** — New(Config) validates required fields (Adapter not nil) and returns (lineage.Service, error), consistent with service constructors across the billing domain. (`func New(config Config) (lineage.Service, error) { if err := config.Validate(); err != nil { return nil, err }; return &service{adapter: config.Adapter}, nil }`)
**Lock-then-mutate segment pattern** — Multi-realization mutations call a locking adapter method (LockCorrectionLineages / LockAdvanceLineagesForBackfill) first to serialize concurrent writers inside the transaction, then perform CloseSegment + CreateSegment splits. (`lineages, err := s.adapter.LockCorrectionLineages(ctx, input.Namespace, correctionOrder)
// then iterate: CloseSegment(id, now) + CreateSegment(remainder) + CreateSegment(covered)`)
**Close-then-split for partial segment amounts** — When a correction or backfill partially covers a segment, the service closes the original and creates two new segments: remainder (same state) and covered amount (new state). This preserves an immutable audit trail. (`s.adapter.CloseSegment(ctx, segment.ID, now)
if remainder.IsPositive() { s.adapter.CreateSegment(ctx, lineage.CreateSegmentInput{Amount: remainder, State: segment.State}) }
s.adapter.CreateSegment(ctx, lineage.CreateSegmentInput{Amount: covered, State: newState})`)
**clock.Now().Truncate(time.Microsecond) for timestamps** — All segment close timestamps use clock.Now().Truncate(time.Microsecond) — never raw time.Now() — to match Postgres microsecond precision and avoid equality check failures. (`now := clock.Now().Truncate(time.Microsecond)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Single file implementing lineage.Service with six methods: CreateInitialLineages, LoadActiveSegmentsByRealizationID, LoadLineagesByCustomer, PersistCorrectionLineageSegments, BackfillAdvanceLineageSegments, CloseSegment, CreateSegment. All writes are transaction-wrapped; reads are straight passthroughs. | PersistCorrectionLineageSegments iterates corrections in a deterministic order (correctionOrder slice) and verifies remaining coverage reaches zero — if a correction exceeds available coverage it returns an error. BackfillAdvanceLineageSegments only covers segments in state LineageSegmentStateAdvanceUncovered. Do not call adapter methods outside the transaction.RunWithNoValue closure. |

## Anti-Patterns

- Bypassing transaction.RunWithNoValue for multi-step adapter calls — creates partial writes when any step fails mid-sequence
- Calling adapter methods outside the transaction.RunWithNoValue closure — the closure's ctx carries the active tx driver; calls outside use a different connection
- Skipping Input.Validate() — lock operations and bulk creates do no further validation inside the transaction
- Using time.Now() directly instead of clock.Now().Truncate(time.Microsecond) for segment timestamps
- Adding Ent query logic directly in service.go — all DB access must go through lineage.Adapter

## Decisions

- **Use transaction.RunWithNoValue at the service layer instead of entutils.TransactingRepo** — The service holds a lineage.Adapter interface (not the concrete *adapter), so it uses the higher-level transaction.Runner abstraction. The adapter implements transaction.Creator, satisfying the runner contract.
- **Lock rows before reading active segments in correction and backfill paths** — Concurrent charge advancement can race to apply corrections against the same realization; locking before reading serializes writers and prevents double-application of the same correction amount.
- **Close-and-split rather than update-in-place for segment mutation** — Segments form an immutable append-only ledger of state transitions. Closing a segment and creating new ones preserves a full audit trail of how each amount was covered or corrected.

## Example: Adding a new service method that locks lineages and transitions segment state

```
import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *service) MyNewMutation(ctx context.Context, input lineage.MyInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
// ...
```

<!-- archie:ai-end -->
