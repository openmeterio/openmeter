# lineage

<!-- archie:ai-start -->

> Domain package for credit realization lineage tracking: defines Service and Adapter interfaces for creating initial lineages, persisting correction segments, and backfilling advance segments. Lineage records link credit realizations to backing transaction groups via append-only Segment rows; mutations use close-then-split semantics rather than in-place updates.

## Patterns

**transaction.RunWithNoValue for multi-step mutations** — Service methods that call multiple adapter methods must wrap them in transaction.RunWithNoValue so all steps share the same ctx-bound transaction. Calls outside the closure use a different connection and bypass the transaction. (`return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error { return s.adapter.CreateLineages(ctx, ...) })`)
**Input.Validate() before any business logic** — Every service method calls input.Validate() as the first step before any adapter or lock operation. This is separate from the adapter-level validation and must not be skipped. (`if err := input.Validate(); err != nil { return err }`)
**Lock-then-mutate segment pattern** — Adapter lock methods (LockCorrectionLineages, LockAdvanceLineagesForBackfill) acquire FOR UPDATE row locks inside an active transaction; must be called before reading segments that will be mutated. (`lineages, err := s.adapter.LockCorrectionLineages(ctx, input.Namespace, realizationIDs)`)
**Close-and-split for partial segment mutation** — When only part of a segment amount is consumed, close the existing segment (CloseSegment) and create two new segments rather than updating the amount in place. Provides an append-only audit trail. (`s.adapter.CloseSegment(ctx, seg.ID, clock.Now().UTC().Truncate(time.Microsecond))
s.adapter.CreateSegment(ctx, CreateSegmentInput{LineageID: seg.LineageID, Amount: consumed, ...})
s.adapter.CreateSegment(ctx, CreateSegmentInput{LineageID: seg.LineageID, Amount: remaining, ...})`)
**clock.Now() with Truncate(time.Microsecond) for segment timestamps** — All timestamps used for segment closed_at must be truncated to microsecond precision to match Postgres storage precision and avoid equality-check failures. Note: this differs from meta package which truncates to streaming.MinimumWindowSizeDuration. (`closedAt := clock.Now().UTC().Truncate(time.Microsecond)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `lineage.go` | Defines Lineage, Segment, ActiveSegmentsByRealizationID, and all input/output types. SortCorrectionPersistSegments and MinDecimal are pure helpers. | BackingTransactionGroupID is required for advance_backfilled and earnings_recognized segments; Segment.Amount must be positive. |
| `service.go` | Service and Adapter interfaces; all input types with Validate(). BackfillAdvanceLineageSegmentsInput requires positive Amount and non-empty BackingTransactionGroupID. | CreateSegmentInput has complex state-dependent validation rules — source_state and backing_transaction_group_id requirements vary by state. |

## Anti-Patterns

- Bypassing transaction.RunWithNoValue for multi-step adapter calls — creates partial writes when any step fails mid-sequence.
- Calling adapter methods outside the transaction.RunWithNoValue closure — the closure's ctx carries the active tx driver; calls outside use a different connection.
- Using clock.Now() without Truncate(time.Microsecond) for segment timestamps — Postgres microsecond precision mismatch causes equality check failures.
- Adding Ent query logic directly in service.go — all DB access must go through lineage.Adapter methods.
- Removing the GetDriverFromContext guard from lock methods in the adapter — makes FOR UPDATE locks unreliable outside explicit transactions.

## Decisions

- **Close-and-split rather than update-in-place for segment mutation** — Append-only segment history enables auditing and prevents lost-update races under concurrent backfill.
- **Lock operations require an active transaction (GetDriverFromContext guard in adapter)** — FOR UPDATE locks are only meaningful inside a transaction; failing fast prevents silent non-locking reads.
- **Package-level LoadActiveSegmentsByRealizationID alongside the adapter method** — Allows cross-package callers (creditpurchase service) to load segments without taking a full lineage.Service dependency.

## Example: Backfill advance lineage segments inside a transaction

```
import (
    "github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
)

err := lineageSvc.BackfillAdvanceLineageSegments(ctx, lineage.BackfillAdvanceLineageSegmentsInput{
    Namespace:                 ns,
    CustomerID:                customerID,
    Currency:                  currency,
    Amount:                    amount,
    BackingTransactionGroupID: txGroupID,
})
```

<!-- archie:ai-end -->
