# creditrealization

<!-- archie:ai-start -->

> Models credit realizations — immutable append-only ledger-linked records tracking which credit amounts have been allocated (positive TypeAllocation) or corrected/reversed (non-positive TypeCorrection) against a charge's service period. Provides Ent mixin, typed Realizations collection with correction-planning and correction-validation logic, and lineage annotations for tracing whether a realization originates from a real credit or an advance.

## Patterns

**Allocation/Correction duality enforced in CreateInput.Validate()** — TypeAllocation requires positive amount and no CorrectsRealizationID. TypeCorrection requires non-positive amount and a non-nil CorrectsRealizationID. Never create a correction without CorrectsRealizationID. (`case TypeAllocation:
    if !i.Amount.IsPositive() { ... }
case TypeCorrection:
    if i.CorrectsRealizationID == nil { ... }`)
**SortHint for deterministic correction ordering** — Realizations created in the same batch must have sequential SortHint values. CreateCorrectionRequest reverses by (CreatedAt DESC, SortHint DESC) so the most-recently-applied allocations are corrected first. (`slices.SortStableFunc(realizations, func(a, b Realization) int {
    cmp := a.CreatedAt.Compare(b.CreatedAt)
    if cmp != 0 { return cmp }
    return a.SortHint - b.SortHint
})`)
**Correction planning via Realizations.Correct** — Use Realizations.Correct(amount, currency, cb) — it plans the CorrectionRequest, validates it, calls the callback for the caller to produce CreateCorrectionInputs, normalizes, re-validates, then returns CreateInputs. Never bypass this pipeline. (`inputs, err := realizations.Correct(amount.Neg(), currency, func(req CorrectionRequest) (CreateCorrectionInputs, error) {
    return mapRequestToInputs(req)
})`)
**Lineage annotations via LineageAnnotations helper** — Set lineage origin using LineageAnnotations(originKind) before persisting an allocation. Read it back with LineageOriginKindFromAnnotations. Never inline the annotation key string. (`annotations := creditrealization.LineageAnnotations(creditrealization.LineageOriginKindRealCredit)`)
**SelfReferenceType required on Mixin instantiation** — creditrealization.Mixin must be instantiated with SelfReferenceType set to the concrete Ent entity pointer type to generate the allocation→corrections self-referencing edge. Omitting it silently drops the edge. (`creditrealization.Mixin{SelfReferenceType: (*entdb.UsageBasedChargeRealization)(nil)}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `models.go` | Core domain types: CreateInput, Type enum, Realization struct, CreateInputs slice with Validate/Sum helpers. Type invariants (positive for allocation, non-positive for correction) enforced here. | Relaxing type invariants here allows corrupt state to reach persistence. |
| `realizations.go` | Realizations collection methods: correction planning (CreateCorrectionRequest), Correct/CorrectAll orchestration, AsCreditsApplied conversion, internal allocationsWithCorrections sort/group. | ErrInsufficientFunds sentinel wraps a GenericValidationError; callers must errors.Is-check it explicitly. |
| `correction.go` | CorrectionRequest / CreateCorrectionInput types and ValidateWith methods that cross-check against existing realizations and total amounts. | ValidateWith mutates a local copy of realizationsWithRemainingAmountByID to track remaining amounts across multiple corrections; the logic breaks if the same allocation appears twice incorrectly. |
| `mixin.go` | Ent mixin with Creator/Getter interfaces and Create/MapFromDB generic helpers for the DB layer. Edges method is on outer Mixin struct, not inner mixinBase. | SelfReferenceType must be set on instantiation; the self-reference edge is conditionally added only if non-nil. |
| `lineage.go` | LineageOriginKind and LineageSegmentState enums plus annotation read/write helpers. | LineageOriginKindFromAnnotations silently skips realizations missing the annotation (no error) in InitialLineageSpecs — intentional for backward compat. |
| `lineage_specs.go` | InitialLineageSpecs derives InitialLineageSpec entries from a Realizations slice for seeding lineage tracking; only processes TypeAllocation entries with a valid lineage annotation. | Uses ulid.Make() for LineageID generation; IDs are not stable across calls. |

## Anti-Patterns

- Creating a TypeCorrection record without going through Realizations.Correct or manually validating CorrectsRealizationID and remaining amount
- Assigning SortHint=0 to all realizations in a batch — ordering becomes undefined for correction planning
- Using amounts without rounding via currencyx.Calculator.RoundToPrecision before creating corrections
- Instantiating creditrealization.Mixin without SelfReferenceType when the entity has self-referencing corrections
- Reading AnnotationLineageOriginKind string key directly instead of using LineageOriginKindFromAnnotations

## Decisions

- **Two-type realization model (allocation + correction) instead of mutable balance rows** — Immutable append-only records provide a full audit trail for credit application and reversal; corrections always reference the allocation they reverse so lineage is unambiguous.
- **Correction planning in pure Go (Realizations.CreateCorrectionRequest) before DB write** — Planning in Go allows dry-run validation, amount splitting across multiple allocations, and unit-testing the correction algorithm without DB involvement.

## Example: Plan and apply corrections for a partial refund

```
import "github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"

calc, _ := currencyx.Code("USD").Calculator()
inputs, err := existingRealizations.Correct(
    alpacadecimal.NewFromFloat(-5),
    calc,
    func(req creditrealization.CorrectionRequest) (creditrealization.CreateCorrectionInputs, error) {
        return buildInputsFromRequest(req, ledgerTxRef)
    },
)
if err != nil {
    return err
}
// persist inputs via adapter
```

<!-- archie:ai-end -->
