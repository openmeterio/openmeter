# creditrealization

<!-- archie:ai-start -->

> Models credit realizations — immutable append-only ledger-linked records tracking which credit amounts have been allocated (positive TypeAllocation) or corrected/reversed (non-positive TypeCorrection) against a charge's service period. Provides an Ent mixin, a typed Realizations collection with correction-planning/validation, and lineage annotations for tracing real-credit vs advance origin.

## Patterns

**Allocation/Correction duality in CreateInput.Validate()** — TypeAllocation requires positive amount and no CorrectsRealizationID. TypeCorrection requires non-positive amount and a non-nil CorrectsRealizationID. Never create a correction without CorrectsRealizationID. (`case TypeAllocation: if !i.Amount.IsPositive() { ... }; case TypeCorrection: if i.CorrectsRealizationID == nil { ... }`)
**SortHint for deterministic correction ordering** — Realizations in the same batch must have sequential SortHint; CreateCorrectionRequest reverses by (CreatedAt DESC, SortHint DESC) so the most-recently-applied allocations are corrected first. (`slices.SortStableFunc(realizations, func(a, b Realization) int { cmp := a.CreatedAt.Compare(b.CreatedAt); if cmp != 0 { return cmp }; return a.SortHint - b.SortHint })`)
**Correction planning via Realizations.Correct** — Use Realizations.Correct(amount, currency, cb) — it plans the CorrectionRequest, validates, invokes the callback to produce CreateCorrectionInputs, normalizes, re-validates, and returns CreateInputs. Never bypass this pipeline. (`inputs, err := realizations.Correct(amount.Neg(), currency, func(req CorrectionRequest) (CreateCorrectionInputs, error) { return mapRequestToInputs(req) })`)
**Lineage annotations via LineageAnnotations helper** — Set lineage origin with LineageAnnotations(originKind) before persisting an allocation; read back with LineageOriginKindFromAnnotations. Never inline the annotation key string. (`annotations := creditrealization.LineageAnnotations(creditrealization.LineageOriginKindRealCredit)`)
**SelfReferenceType required on Mixin instantiation** — creditrealization.Mixin must be instantiated with SelfReferenceType set to the concrete Ent entity pointer to generate the allocation->corrections self-referencing edge; omitting it silently drops the edge. (`creditrealization.Mixin{SelfReferenceType: (*entdb.UsageBasedChargeRealization)(nil)}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `models.go` | Core domain types: CreateInput, Type enum, Realization, CreateInputs slice with Validate/Sum. Type invariants (positive allocation, non-positive correction) live here. | Relaxing type invariants here lets corrupt state reach persistence. |
| `realizations.go` | Realizations collection methods: CreateCorrectionRequest, Correct/CorrectAll, AsCreditsApplied, internal allocationsWithCorrections sort/group. | ErrInsufficientFunds wraps a GenericValidationError; callers must errors.Is-check it explicitly. |
| `correction.go` | CorrectionRequest / CreateCorrectionInput with ValidateWith methods cross-checking existing realizations and totals. | ValidateWith mutates a local copy of realizationsWithRemainingAmountByID to track remaining amounts; logic breaks if the same allocation appears twice incorrectly. |
| `mixin.go` | Ent mixin with Creator/Getter interfaces and Create/MapFromDB helpers. Edges method is on the outer Mixin struct, not inner mixinBase. | SelfReferenceType must be set on instantiation; the self-reference edge is added only when non-nil. |
| `lineage.go` | LineageOriginKind and LineageSegmentState enums plus annotation read/write helpers. | LineageOriginKindFromAnnotations silently skips realizations missing the annotation (no error) — intentional for backward compat. |
| `lineage_specs.go` | InitialLineageSpecs derives InitialLineageSpec entries from a Realizations slice for seeding lineage; only processes TypeAllocation with a valid lineage annotation. | Uses ulid.Make() for LineageID — IDs are not stable across calls. |

## Anti-Patterns

- Creating a TypeCorrection without going through Realizations.Correct or validating CorrectsRealizationID and remaining amount
- Assigning SortHint=0 to all realizations in a batch — ordering becomes undefined for correction planning
- Using amounts without rounding via currencyx.Calculator.RoundToPrecision before creating corrections
- Instantiating creditrealization.Mixin without SelfReferenceType when the entity has self-referencing corrections
- Reading the AnnotationLineageOriginKind string key directly instead of LineageOriginKindFromAnnotations

## Decisions

- **Two-type realization model (allocation + correction) instead of mutable balance rows** — Immutable append-only records provide a full audit trail; corrections always reference the allocation they reverse so lineage is unambiguous.
- **Correction planning in pure Go before DB write** — Planning in Go allows dry-run validation, amount splitting across allocations, and unit-testing the correction algorithm without a DB.

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
