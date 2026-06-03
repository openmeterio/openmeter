# lineage

<!-- archie:ai-start -->

> Domain package for credit realization lineage tracking: defines Service and Adapter interfaces and the append-only Lineage/Segment model linking credit realizations to backing ledger transaction groups. Children split persistence (adapter/, Ent + FOR UPDATE locks) from orchestration (service/, transaction.RunWithNoValue). Primary constraint: segment mutation is close-then-split, never in-place update.

## Patterns

**transaction.RunWithNoValue for multi-step mutations** — Service methods calling multiple adapter methods wrap them in transaction.RunWithNoValue so all steps share the ctx-bound transaction. Calls outside the closure use a different connection and bypass the tx. (`return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error { return s.adapter.CreateLineages(ctx, in) })`)
**Input.Validate() before any business logic** — Every service method calls input.Validate() first; this is separate from adapter-level validation and must not be skipped because lock/bulk-create paths do no further validation inside the transaction. (`if err := input.Validate(); err != nil { return err }`)
**Lock-then-mutate via FOR UPDATE** — Adapter lock methods (LockCorrectionLineages, LockAdvanceLineagesForBackfill) take FOR UPDATE row locks inside an active tx and must be called before reading segments that will be mutated; the adapter guards with GetDriverFromContext. (`lineages, err := s.adapter.LockCorrectionLineages(ctx, ns, realizationIDs)`)
**Close-and-split for partial segment mutation** — When only part of a segment amount is consumed, CloseSegment the existing row and CreateSegment two new rows rather than updating in place — append-only audit trail. (`s.adapter.CloseSegment(ctx, seg.ID, closedAt); s.adapter.CreateSegment(ctx, consumedInput); s.adapter.CreateSegment(ctx, remainingInput)`)
**clock.Now().Truncate(time.Microsecond) for segment timestamps** — All closed_at timestamps truncate to microsecond precision to match Postgres storage and avoid equality-check failures. Note this differs from the meta package, which truncates to streaming.MinimumWindowSizeDuration. (`closedAt := clock.Now().UTC().Truncate(time.Microsecond)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `lineage.go` | Segment.Validate() with state-dependent rules, SortCorrectionPersistSegments and MinDecimal pure helpers. | BackingTransactionGroupID required for advance_backfilled and earnings_recognized segments; Segment.Amount must be positive. |
| `service.go` | Service and Adapter interfaces plus all input types with Validate(). Lineage/Segment/ActiveSegmentsByRealizationID types. | CreateSegmentInput validation is state-dependent (source_state / backing_transaction_group_id requirements vary by State). |
| `lineage_test.go` | Validates the source-backing-transaction-group rule for earnings_recognized segments sourced from advance_backfilled. | Mirror this table-driven validation style when adding new state combinations. |

## Anti-Patterns

- Bypassing transaction.RunWithNoValue for multi-step adapter calls — creates partial writes when a step fails mid-sequence.
- Calling adapter methods outside the RunWithNoValue closure — the closure's ctx carries the active tx; outside calls use a different connection.
- Using clock.Now() without Truncate(time.Microsecond) for segment timestamps — Postgres precision mismatch fails equality checks.
- Adding Ent query logic directly in service.go — all DB access goes through lineage.Adapter.
- Removing the GetDriverFromContext guard from adapter lock methods — makes FOR UPDATE unreliable outside transactions.

## Decisions

- **Close-and-split rather than update-in-place for segment mutation.** — Append-only segment history enables auditing and prevents lost-update races under concurrent backfill.
- **Lock operations require an active transaction (GetDriverFromContext guard).** — FOR UPDATE locks are only meaningful inside a transaction; failing fast prevents silent non-locking reads.
- **Package-level LoadActiveSegmentsByRealizationID alongside the adapter method.** — Lets cross-package callers (creditpurchase service) load segments without a full lineage.Service dependency.

## Example: Backfill advance lineage segments inside a transaction

```
import "github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"

err := lineageSvc.BackfillAdvanceLineageSegments(ctx, lineage.BackfillAdvanceLineageSegmentsInput{
  Namespace: ns, CustomerID: customerID, Currency: currency,
  Amount: amount, BackingTransactionGroupID: txGroupID,
})
```

<!-- archie:ai-end -->
