# service

<!-- archie:ai-start -->

> Business-logic service implementing lineage.Service for credit realization lineage lifecycle: creating initial lineages, persisting correction segments, and backfilling advance segments. Orchestrates multi-step segment mutation sequences within transactions, delegating all persistence to lineage.Adapter.

## Patterns

**transaction.RunWithNoValue for all multi-step mutations** — Every method that modifies segments uses transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error { ... }) to wrap the full operation in a single atomic transaction, ensuring partial writes don't occur. (`return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error { ...; return s.adapter.CreateLineages(ctx, ...) })`)
**Input.Validate() before any business logic** — Every public method calls input.Validate() as the first operation and returns early on validation failure, before acquiring any locks or performing DB reads. (`func (s *service) CreateInitialLineages(ctx context.Context, input lineage.CreateInitialLineagesInput) error { if err := input.Validate(); err != nil { return err }; ... }`)
**Config struct + Validate() + New() constructor** — New(Config) validates required fields (Adapter not nil) and returns (lineage.Service, error), consistent with service constructors across the billing domain. (`func New(config Config) (lineage.Service, error) { if err := config.Validate(); err != nil { return nil, err }; return &service{adapter: config.Adapter}, nil }`)
**Lock-then-mutate segment pattern** — Multi-realization mutations (PersistCorrectionLineageSegments, BackfillAdvanceLineageSegments) call a locking adapter method (LockCorrectionLineages / LockAdvanceLineagesForBackfill) first to serialize concurrent writers, then perform CloseSegment + CreateSegment splits inside the same transaction. (`lineages, err := s.adapter.LockCorrectionLineages(ctx, input.Namespace, correctionOrder)
// then iterate: CloseSegment(id, now) + CreateSegment(remainder) + CreateSegment(backfilled)`)
**Close-then-split segment approach for partial amounts** — When a correction or backfill only partially covers a segment, the service closes the original segment and creates two new segments: one for the remainder (unchanged state) and one for the covered amount (new state), keeping amounts fully accounted. (`s.adapter.CloseSegment(ctx, segment.ID, now)
if remainder.IsPositive() { s.adapter.CreateSegment(ctx, lineage.CreateSegmentInput{Amount: remainder, State: segment.State}) }
s.adapter.CreateSegment(ctx, lineage.CreateSegmentInput{Amount: coveredAmount, State: newState})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Single file implementing lineage.Service with four methods: CreateInitialLineages, LoadActiveSegmentsByRealizationID, PersistCorrectionLineageSegments, BackfillAdvanceLineageSegments. All writes are transaction-wrapped; read delegation (LoadActiveSegmentsByRealizationID) is a straight passthrough. | PersistCorrectionLineageSegments iterates corrections in a deterministic order (correctionOrder slice) and verifies remaining coverage reaches zero — if a correction exceeds available lineage coverage it returns an error instead of silently dropping amounts. clock.Now().Truncate(time.Microsecond) is used for segment timestamps — do not use time.Now() directly. |

## Anti-Patterns

- Bypassing transaction.RunWithNoValue for multi-step adapter calls — creates partial writes when any step fails mid-sequence
- Calling adapter methods outside the transaction.RunWithNoValue closure — the closure's ctx carries the active tx driver; calls outside use a different connection
- Skipping Input.Validate() — lock operations and bulk creates do no further validation once inside the transaction
- Calling clock.Now() without Truncate(time.Microsecond) for segment timestamps — Postgres microsecond precision mismatch causes equality check failures
- Adding Ent query logic directly in service.go — all DB access must go through lineage.Adapter methods

## Decisions

- **Use transaction.RunWithNoValue instead of entutils.TransactingRepoWithNoValue at the service layer** — The service holds an Adapter interface (not a concrete *adapter), so it uses the higher-level transaction.Runner abstraction. The adapter implements transaction.Creator, satisfying the runner contract.
- **Correction and backfill paths lock rows before reading them** — Concurrent charge advancement can race to apply corrections against the same realization; locking before reading the active segments serializes writers and prevents double-application of the same correction amount.
- **Close-and-split rather than update-in-place for segment mutation** — Segments form an immutable append-only ledger of state transitions. Closing a segment and creating new ones preserves a full audit trail of how each amount was covered or corrected.

## Example: Adding a new service method that locks lineages and creates a new segment type

```
import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *service) MyNewMutation(ctx context.Context, input lineage.MyInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

// ...
```

<!-- archie:ai-end -->
