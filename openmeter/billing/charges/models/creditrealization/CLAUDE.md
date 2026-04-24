# creditrealization

<!-- archie:ai-start -->

> Models credit realizations — the ledger-linked records that track which credit amounts have been allocated (positive) or corrected/reversed (non-positive) against a charge's service period. Provides Ent mixin for persistence, a typed Realizations collection with correction-planning and correction-validation logic, and lineage annotations for tracing whether a realization originates from a real credit or an advance.

## Patterns

**Allocation/Correction duality** — Every Realization has a Type field: TypeAllocation (positive amount, no CorrectsRealizationID) or TypeCorrection (non-positive amount, CorrectsRealizationID required). CreateInput.Validate() enforces these invariants. Never create a correction without a non-empty CorrectsRealizationID. (`switch i.Type {
case TypeAllocation:
    if !i.Amount.IsPositive() { ... }
case TypeCorrection:
    if i.CorrectsRealizationID == nil { ... }
}`)
**SortHint for deterministic correction ordering** — Realizations created in the same batch must be assigned sequential SortHint values. CreateCorrectionRequest reverts in reverse (CreatedAt DESC, SortHint DESC) order so the most-recently-applied allocations are corrected first. (`slices.SortStableFunc(realizations, func(a, b Realization) int {
    cmpCreatedAt := a.CreatedAt.Compare(b.CreatedAt)
    if cmpCreatedAt != 0 { return cmpCreatedAt }
    return a.SortHint - b.SortHint
})`)
**Correction planning via Realizations.Correct** — Use Realizations.Correct(amount, currency, cb) — it plans the CorrectionRequest, validates it, calls the callback for the caller to produce CreateCorrectionInputs, normalizes, re-validates, then returns CreateInputs. Never bypass this pipeline to create correction records directly. (`inputs, err := realizations.Correct(amount.Neg(), currency, func(req CorrectionRequest) (CreateCorrectionInputs, error) {
    return mapRequestToInputs(req)
})`)
**Lineage annotations via LineageAnnotations helper** — Set lineage origin on an allocation's Annotations using LineageAnnotations(originKind) before persisting. Read it back with LineageOriginKindFromAnnotations. Never store the annotation key string inline. (`annotations := creditrealization.LineageAnnotations(creditrealization.LineageOriginKindRealCredit)`)
**Self-referencing edge via Mixin.SelfReferenceType** — The creditrealization.Mixin must be instantiated with SelfReferenceType set to the concrete Ent entity type to generate the allocation→corrections edge. Omitting it silently drops the self-reference edge from the schema. (`creditrealization.Mixin{SelfReferenceType: (*entdb.UsageBasedChargeRealization)(nil)}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `models.go` | Core domain types: CreateInput, Type enum, Realization struct, CreateInputs slice with Validate/Sum helpers. | Type invariants (positive for allocation, non-positive for correction) are validated here; relax them and persistence will accept corrupt state. |
| `realizations.go` | Realizations collection methods: correction planning (CreateCorrectionRequest), Correct/CorrectAll orchestration, AsCreditsApplied conversion, internal allocationsWithCorrections sort/group. | ErrInsufficientFunds sentinel wraps a GenericValidationError; callers should errors.Is-check it explicitly. |
| `correction.go` | CorrectionRequest / CreateCorrectionInput types and their ValidateWith methods that cross-check against existing realizations and total amounts. | ValidateWith mutates a local copy of realizationsWithRemainingAmountByID to track remaining amounts across multiple corrections for the same allocation; the logic breaks if the same allocation appears twice in a single correction batch incorrectly. |
| `mixin.go` | Ent mixin with Creator/Getter interfaces and Create/MapFromDB generic helpers for the DB layer. | SelfReferenceType must be set on instantiation; the edges method is on the outer Mixin struct, not the inner mixinBase. |
| `lineage.go` | LineageOriginKind and LineageSegmentState enums plus annotation read/write helpers for tracking credit origin. | LineageOriginKindFromAnnotations silently skips realizations missing the annotation (no error) in InitialLineageSpecs — intended for backward compat with unrealized credits. |
| `lineage_specs.go` | InitialLineageSpecs derives InitialLineageSpec entries from a Realizations slice for seeding lineage tracking; only processes TypeAllocation entries with a valid lineage annotation. | Uses ulid.Make() for LineageID generation; IDs are not stable across calls. |

## Anti-Patterns

- Creating a TypeCorrection record without going through Realizations.Correct or manually validating CorrectsRealizationID and remaining amount
- Assigning SortHint=0 to all realizations in a batch — ordering becomes undefined for correction planning
- Using amounts without rounding via currencyx.Calculator.RoundToPrecision before creating corrections
- Instantiating creditrealization.Mixin without SelfReferenceType when the entity has self-referencing corrections
- Directly reading AnnotationLineageOriginKind string key instead of using LineageOriginKindFromAnnotations

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
