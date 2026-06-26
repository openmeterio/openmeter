# lineage

<!-- archie:ai-start -->

> Domain package defining the credit-realization lineage model — the append-only audit trail tracking how each charge's realized value moves through states (real_credit/advance_uncovered -> advance_backfilled -> earnings_recognized) and is consumed during corrections. Declares the Service and Adapter interfaces plus the Lineage/Segment value types and pure sorting/matching helpers.

## Patterns

**Service/Adapter split with shared method names** — Service is business orchestration (CreateInitialLineages, PersistCorrectionLineageSegments, BackfillAdvanceLineageSegments, Close/CreateSegment); Adapter (entutils.TxCreator) is persistence with Lock* + Load* + List* primitives the service composes (`Service.PersistCorrectionLineageSegments composes Adapter.LockCorrectionLineages + CreateSegment`)
**State-precedence sort for correction consumption** — SortCorrectionPersistSegments orders segments by state precedence (earnings_recognized<advance_backfilled<advance_uncovered<real_credit), not creation order, so corrections drain in the intended priority (`sort.SliceStable on precedence(state)`)
**Segment.Validate enforces state-conditional fields** — advance_backfilled and earnings_recognized require BackingTransactionGroupID; earnings_recognized requires a SourceState that is not itself earnings_recognized, plus SourceBackingTransactionGroupID when SourceState is advance_backfilled (`Segment.Validate / CreateSegmentInput.Validate`)
**Feature-filter advance matching** — FeatureFiltersMatchAdvance: empty filters match all; non-empty filters match only when at least one advance feature intersects; FilterAdvanceLineagesForBackfill applies it across lineages (`FeatureFiltersMatchAdvance(filters, lineage.AdvanceFeatures)`)
**Inputs validate before mutation** — Every *Input has Validate() returning errors.Join(errs...); PersistCorrectionLineageSegmentsInput specifically requires CorrectsRealizationID on TypeCorrection realizations (`CreateInitialLineagesInput.Validate, BackfillAdvanceLineageSegmentsInput.Validate`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service + Adapter interfaces, all *Input structs, and the Lineage/Segment/ActiveSegmentsByRealizationID types | Adapter Lock* methods (LockCorrectionLineages, LockAdvanceLineagesForBackfill) imply row locks that must run inside a transaction; CreateSegmentInput.Validate carries the full state-machine field rules |
| `lineage.go` | Pure helpers: SortCorrectionPersistSegments, MinDecimal, FilterAdvanceLineagesForBackfill, FeatureFiltersMatchAdvance, and Segment.Validate | Segment.Validate uses errors.Join directly (not NewNillableGenericValidationError); amount must be positive |
| `lineage_test.go` | Unit tests for Segment.Validate source-backing rules and FeatureFiltersMatchAdvance truth table | Encodes the exact error string contract for advance_backfilled source segments |

## Anti-Patterns

- Mutating a segment's amount in place instead of Close + Create-remainder — breaks the append-only audit model (the service does this; never bypass it)
- Consuming correction segments in raw creation order instead of SortCorrectionPersistSegments precedence
- Creating an earnings_recognized segment without a non-recognized SourceState (and SourceBackingTransactionGroupID when source is advance_backfilled)
- Treating empty featureFilters as matching nothing — empty filters match all advances
- Adding Ent queries to the service/value types — persistence must go through Adapter

## Decisions

- **Segments are append-only; consumption closes and creates a remainder** — Preserves a verifiable audit trail of how realized value was covered/corrected/backfilled over time
- **Correction consumption ordered by state precedence, not creation order** — Earnings-recognized and backfilled value must be drained before raw real_credit to keep recognition accounting correct

## Example: Matching advance features against a lineage during backfill

```
func FeatureFiltersMatchAdvance(featureFilters []string, advanceFeatures []string) bool {
  if len(featureFilters) == 0 { return true }
  if len(advanceFeatures) == 0 { return false }
  for _, feature := range advanceFeatures {
    if lo.Contains(featureFilters, feature) { return true }
  }
  return false
}
```

<!-- archie:ai-end -->
