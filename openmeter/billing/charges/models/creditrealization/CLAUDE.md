# creditrealization

<!-- archie:ai-start -->

> Domain models + Ent mixin for credit realizations: positive 'allocation' entries and non-positive 'correction' entries that record how credit/advance value was applied to charges and invoice lines. Holds the core correction-planning algorithm (reverse-order draining of remaining allocation amounts) and the lineage state machine that tracks value as it moves from real_credit/advance through backfill to earnings recognition.

## Patterns

**Allocation/correction polymorphism via Type enum** — A Realization is either TypeAllocation (positive Amount) or TypeCorrection (non-positive Amount, requires CorrectsRealizationID). CreateInput.Validate switches on Type to enforce sign and required fields. (`case TypeCorrection: if i.Amount.IsPositive() { errs = append(errs, ...) }`)
**Currency-aware validation and rounding** — Correction inputs and requests validate/normalize against a currencyx.Calculator: amounts must be rounded to the smallest denomination (IsRoundedToPrecision) and normalized via RoundToPrecision before planning. (`func (i CorrectionRequestItem) ValidateWith(currency currencyx.Calculator) error`)
**Reverse-order correction draining** — CreateCorrectionRequest reverses allocations (sorted by CreatedAt then SortHint), drains each allocation's RemainingAmount until the requested negative amount is satisfied, returning ErrInsufficientFunds if it cannot. (`mutable.Reverse(allocationsWithCorrections); ... return CorrectionRequest{}, ErrInsufficientFunds`)
**Lineage state machine via annotations** — Lineage origin (real_credit/advance) is stored as a models.Annotations key (AnnotationLineageOriginKind); InitialLineageSegmentState maps origin to the starting LineageSegmentState which then transitions toward earnings_recognized. (`func InitialLineageSegmentState(originKind LineageOriginKind) LineageSegmentState`)
**Collection-typed helpers with Validate/Sum** — Realizations, CreateInputs, CreateAllocationInputs, CreateCorrectionInputs are named slice types carrying Validate() (joining indexed errors) and Sum() (alpacadecimal aggregation) methods. (`func (i CreateInputs) Validate() error { ... fmt.Errorf("credit realization input[%d]: %w", idx, err) }`)
**Self-referencing Ent edge for corrections** — The Mixin optionally wires an edge.To("allocation").Field("corrects_realization_id").Unique().From("corrections") only when SelfReferenceType is set, so a correction points back to its allocation in the same table. (`edge.To("allocation", m.SelfReferenceType).Field("corrects_realization_id").Unique().From("corrections")`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `realizations.go` | Core algorithm: Realizations slice with CreateCorrectionRequest, Correct/CorrectAll (callback-driven), AsCreditsApplied, and the private allocationsWithCorrections that computes remaining amounts. | allocationsWithCorrections sorts by CreatedAt then SortHint; corrections must reference a known allocation or it errors. RemainingAmount going negative is a corrupt-state error. |
| `models.go` | CreateInput, Type enum, Realization struct (embeds CreateInput + Namespaced/ManagedModel + SortHint), per-type validation. | SortHint encodes batch priority order; reverts depend on reverse iteration of it. |
| `correction.go` | CorrectionRequest/CreateCorrectionInputs validation against existing realizations and a total-to-correct amount, plus AsCreateInputs conversion. | ValidateWith re-derives remaining amounts and breaks early on first corrupt reference to avoid cascading false errors. |
| `allocation.go` | CreateAllocationInput(s) with Validate/Sum and AsCreateInputs (sets Type=TypeAllocation). | Amount must be strictly positive for allocations. |
| `lineage.go` | LineageOriginKind / LineageSegmentState enums, annotation read/write helpers, InitialLineageSegmentState mapping. | Segment states (real_credit, advance_uncovered, advance_backfilled, earnings_recognized) are a directed lifecycle; don't add states without updating Values()/Validate(). |
| `lineage_specs.go` | InitialLineageSpecs builds per-allocation lineage seeds (ulid LineageID, root realization, origin, initial state, amount). | Skips non-allocation realizations and silently skips allocations lacking the origin-kind annotation. |
| `mixin.go` | Ent schema mixin (amount numeric, ledger_transaction_group_id immutable, type enum, optional corrects_realization_id) + generic Create/MapFromDB. | ledger_transaction_group_id is Immutable; the self-edge only materializes when SelfReferenceType is supplied by the embedding schema. |
| `correction_test.go` | Table of subtests exercising correction planning, reverse-order draining, rounding, and ErrInsufficientFunds boundaries. | Uses InexactFloat64() float assertions and USD calculator helper; mirror this style for new cases. |

## Anti-Patterns

- Creating a TypeCorrection without CorrectsRealizationID or with a positive amount.
- Comparing/aggregating decimal amounts without rounding to currency precision first.
- Iterating allocations in forward order when reverting — reverts must drain in reverse (CreatedAt then SortHint).
- Adding a LineageSegmentState or LineageOriginKind value without updating its Values()/Validate().
- Mutating ledger_transaction_group_id after creation — it is immutable in the schema.

## Decisions

- **Corrections are separate non-positive rows referencing their allocation, not in-place edits** — Preserves an auditable ledger-like history of how much of each allocation remains and allows reverse-order reversal across batches.
- **Lineage origin/state carried as annotations + explicit segment table state** — Lets advance-backed usage be backfilled by later credit purchases and tracked through to earnings recognition without losing provenance.
- **Validation/rounding always goes through currencyx.Calculator** — Money correctness requires consistent smallest-denomination rounding; passing the calculator in keeps the models currency-agnostic.

## Example: Planning a negative correction by draining allocations in reverse order

```
func (r Realizations) CreateCorrectionRequest(amount alpacadecimal.Decimal, currency currencyx.Calculator) (CorrectionRequest, error) {
	if amount.IsPositive() { return CorrectionRequest{}, models.NewGenericValidationError(errors.New("amount must not be positive")) }
	amount = currency.RoundToPrecision(amount)
	allocationsWithCorrections, err := r.allocationsWithCorrections()
	if err != nil { return CorrectionRequest{}, err }
	mutable.Reverse(allocationsWithCorrections)
	// drain RemainingAmount per allocation until amountToCorrect is satisfied, else ErrInsufficientFunds
}
```

<!-- archie:ai-end -->
