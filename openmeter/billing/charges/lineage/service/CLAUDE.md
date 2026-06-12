# service

<!-- archie:ai-start -->

> Service layer implementing lineage.Service — the business logic that builds initial credit-realization lineages, FIFO-consumes active segments for corrections, and backfills advance-uncovered segments. Pure orchestration over the lineage.Adapter; all persistence is delegated.

## Patterns

**Config{Adapter lineage.Adapter} + New returns lineage.Service** — Constructor validates Adapter != nil and returns the lineage.Service interface (from ../lineage/service.go). service struct holds only the adapter. (`func New(config Config) (lineage.Service, error) { if err := config.Validate(); err != nil { return nil, err }; return &service{adapter: config.Adapter}, nil }`)
**Multi-step methods wrapped in transaction.RunWithNoValue(ctx, s.adapter, ...)** — Any method that locks then mutates (CreateInitialLineages, PersistCorrectionLineageSegments, BackfillAdvanceLineageSegments) runs inside transaction.RunWithNoValue so the Lock*/Close/Create calls share one tx. Pass s.adapter as the TxCreator. (`return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error { ... })`)
**Validate input at the top of every mutating method** — Each method calls input.Validate() (defined on the input structs in lineage/service.go) before doing work and returns early on error. (`if err := input.Validate(); err != nil { return err }`)
**Append-only segment consumption: Close + Create remainder** — To consume part of a segment, compute consumedAmount := lineage.MinDecimal(segment.Amount, remaining), CloseSegment(segment.ID, now), and if a remainder is positive CreateSegment a new segment carrying the same state/backing IDs. Never mutate amounts in place. (`if err := s.adapter.CloseSegment(ctx, segment.ID, now); err != nil { ... }; remainder := segment.Amount.Sub(consumedAmount); if remainder.IsPositive() { s.adapter.CreateSegment(ctx, lineage.CreateSegmentInput{...}) }`)
**Deterministic ordering via lineage helpers** — Correction consumption iterates lineage.SortCorrectionPersistSegments(entry.Segments) (state precedence: earnings_recognized < advance_backfilled < advance_uncovered < real_credit). Backfill filters via lineage.FilterAdvanceLineagesForBackfill(lineages, input.FeatureFilters). (`for _, segment := range lineage.SortCorrectionPersistSegments(entry.Segments) { ... }`)
**Timestamps truncated to microsecond from clock.Now()** — now := clock.Now().Truncate(time.Microsecond) before closing segments, so closed_at aligns with Postgres microsecond precision and stays test-freezable via pkg/clock. (`now := clock.Now().Truncate(time.Microsecond)`)
**Coverage shortfall is a hard error** — If correction remaining stays positive after walking all segments, the method returns an error ('correction amount %s exceeds active lineage coverage ...') — over-consumption is never silently allowed. (`if remaining.IsPositive() { return fmt.Errorf("correction amount %s exceeds active lineage coverage for realization %s", remaining.String(), realizationID) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Whole lineage.Service implementation: CreateInitialLineages (builds specs via creditrealization.InitialLineageSpecs, tags Advance specs with input.Features), PersistCorrectionLineageSegments (FIFO close/split against correction amounts), BackfillAdvanceLineageSegments (covers advance_uncovered with a backing tx group), plus thin CloseSegment/CreateSegment pass-throughs. | Backfill splits a partially-covered segment into three writes: close original, create advance_uncovered remainder, create advance_backfilled covered piece with BackingTransactionGroupID. CreateInitialLineages only sets AdvanceFeatures on specs whose OriginKind == LineageOriginKindAdvance. CloseSegment/CreateSegment delegate directly without their own transaction wrapper — callers must supply one if atomicity across calls is needed. |

## Anti-Patterns

- Mutating a segment's amount in place instead of Close + Create-remainder — breaks the append-only audit model.
- Calling adapter Lock*/Close/Create across multiple statements without an enclosing transaction.RunWithNoValue — partial failures would corrupt coverage.
- Using time.Now() instead of clock.Now().Truncate(time.Microsecond) — defeats test time-freezing and risks sub-microsecond drift vs Postgres.
- Swallowing the positive-remaining coverage shortfall instead of returning the over-consumption error.
- Adding Ent queries here — the service must go through lineage.Adapter, never touch *entdb.Client directly.

## Decisions

- **Correction segments are consumed in SortCorrectionPersistSegments precedence order, not raw creation order.** — Recognized earnings and backfilled advances must be unwound before uncovered advances and real credit, so corrections reverse value in the economically correct sequence.
- **All multi-write methods share a single transaction via transaction.RunWithNoValue.** — Lock-then-mutate flows (corrections, backfill) must be atomic so concurrent realization runs cannot double-consume the same segment.

## Example: FIFO-consuming active segments for a correction inside one transaction

```
return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
	lineages, err := s.adapter.LockCorrectionLineages(ctx, input.Namespace, correctionOrder)
	if err != nil { return fmt.Errorf("lock lineages for correction persistence: %w", err) }
	now := clock.Now().Truncate(time.Microsecond)
	for _, realizationID := range correctionOrder {
		entry := lineagesByRealizationID[realizationID]
		remaining := correctionAmountsByRealizationID[realizationID]
		for _, segment := range lineage.SortCorrectionPersistSegments(entry.Segments) {
			if !remaining.IsPositive() { break }
			consumed := lineage.MinDecimal(segment.Amount, remaining)
			if err := s.adapter.CloseSegment(ctx, segment.ID, now); err != nil { return err }
			if rem := segment.Amount.Sub(consumed); rem.IsPositive() {
				_ = s.adapter.CreateSegment(ctx, lineage.CreateSegmentInput{LineageID: segment.LineageID, Amount: rem, State: segment.State, BackingTransactionGroupID: segment.BackingTransactionGroupID})
			}
			remaining = remaining.Sub(consumed)
// ...
```

<!-- archie:ai-end -->
