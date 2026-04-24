# invoicecalc

<!-- archie:ai-start -->

> Pure, stateless invoice calculation pipeline for the billing domain. Applies ordered sequences of transformation functions to StandardInvoice and GatheringInvoice objects to compute derived fields (collection dates, draft/due dates, service periods, discount correlation IDs, detailed line totals, and tax config snapshots) without touching the database.

## Patterns

**Typed calculation function signatures** — Every calculation step is a function matching `StandardInvoiceCalculation = func(*billing.StandardInvoice, StandardInvoiceCalculatorDependencies) error` or `GatheringInvoiceCalculation = func(*billing.GatheringInvoice, GatheringInvoiceCalculatorDependencies) error`. New calculations must match one of these two signatures. (`func MyCalc(inv *billing.StandardInvoice, deps StandardInvoiceCalculatorDependencies) error { ... }`)
**Register calculations in InvoiceCalculations registry** — All active calculation steps are registered in the `InvoiceCalculations` package-level var in calculator.go under the correct slice: `Invoice`, `GatheringInvoice`, or `GatheringInvoiceWithLiveData`. Steps run in declaration order; order matters for dependent computations (e.g. CollectionAt must precede DraftUntil). (`InvoiceCalculations.Invoice = []StandardInvoiceCalculation{ WithNoDependencies(StandardInvoiceCollectionAt), WithNoDependencies(CalculateDraftUntil), RecalculateDetailedLinesAndTotals, SnapshotTaxConfigIntoLines }`)
**WithNoDependencies / WithNoGatheringDependencies wrappers** — Use `WithNoDependencies` to wrap a `func(*billing.StandardInvoice) error` into a `StandardInvoiceCalculation` when the step does not need `StandardInvoiceCalculatorDependencies`. Use `WithNoGatheringDependencies` for the gathering counterpart. Do not ignore the deps parameter in a raw function — wrap it instead. (`WithNoDependencies(CalculateDraftUntil)`)
**Error aggregation via MergeValidationIssues** — In `applyCalculations`, all step errors are joined with `errors.Join` and then funnelled through `invoice.MergeValidationIssues(billing.ValidationWithComponent(...), component)`. New calculation steps must return plain errors; the pipeline wraps them into ValidationIssues. GatheringInvoice calculations return raw joined errors because GatheringInvoice has no ValidationIssues field. (`return invoice.MergeValidationIssues(billing.ValidationWithComponent(billing.ValidationComponentOpenMeter, outErr), billing.ValidationComponentOpenMeter)`)
**Require expanded lines before operating on them** — Every calculation that iterates lines must guard with `i.Lines.IsPresent()` (or equivalent) and return an explicit error when absent. Never silently skip or operate on an absent `Lines` optional. (`if !i.Lines.IsPresent() { return errors.New("lines must be expanded") }`)
**LineEngine dispatch via LineEngineResolver** — `RecalculateDetailedLinesAndTotals` groups lines by `line.Engine` and calls `deps.LineEngines.Get(lineEngineType)` to dispatch to the correct engine. New calculation steps that need per-engine dispatch must accept `LineEngineResolver` through `StandardInvoiceCalculatorDependencies`, not hardcode engine-specific logic. (`lineEngine, err := deps.LineEngines.Get(lineEngineType); updatedLines, err := lineEngine.CalculateLines(input)`)
**TaxCodes pre-resolved map pattern** — `SnapshotTaxConfigIntoLines` reads from `deps.TaxCodes TaxCodes` (a `map[string]taxcode.TaxCode`) that must be pre-populated before the calculator runs. The map is a value type with a `Get(stripeCode string) (*taxcode.TaxCode, bool)` helper. New tax-related calculations must read from this pre-resolved map, not query the DB directly. (`tc, ok := deps.TaxCodes.Get(line.TaxConfig.Stripe.Code)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `calculator.go` | Defines `Calculator` interface, `invoiceCalculatorsByType` registry, `StandardInvoiceCalculatorDependencies`, `GatheringInvoiceCalculatorDependencies`, `LineEngineResolver`, `TaxCodes`, and `New()`. Entry point for callers. `applyCalculations` is the internal loop that runs all steps and merges errors. | Step registration order in `InvoiceCalculations` matters — CollectionAt must precede DraftUntil which must precede DueAt. Adding a step without checking ordering can silently produce wrong derived fields. |
| `details.go` | Contains `RecalculateDetailedLinesAndTotals` (engine dispatch + totals rollup) and `MergeGeneratedDetailedLines` (materialises rating.GenerateDetailedLinesResult into a parent StandardLine). Called by the calculator pipeline and by the rating integration. | `ValidateStandardLineIDsMatchExactly` enforces that engines return exactly the same line IDs they received. Engines that create or drop lines will cause a hard error here. |
| `collectionat.go` | Computes `CollectionAt` for StandardInvoice and `NextCollectionAt` for GatheringInvoice. Handles subscription-aligned and anchored-aligned collection configs using `timeutil.NewRecurrenceFromISODuration`. | Only metered lines (those where `line.DependsOnMeteredQuantity()` is true) contribute to StandardInvoice collection time; flat-fee lines are explicitly excluded. Deleted lines are excluded via nil-check on `DeletedAt`. |
| `taxconfig.go` | Merges `invoice.Workflow.Config.Invoicing.DefaultTaxConfig` into each line via `productcatalog.MergeTaxConfigs`, then stamps resolved `TaxCode` entity and `TaxCodeID` from the pre-resolved `deps.TaxCodes` map. Is a no-op for gathering invoices. | Does not overwrite an existing `TaxCodeID` (preserves caller-set IDs); always stamps `TaxCode` entity if found. Skip logic: step returns nil immediately when `invoice.Status == billing.StandardInvoiceStatusGathering`. |
| `mock.go` | Provides `MockableInvoiceCalculator` (wraps real Calculator + optional mockCalculator) for test injection. `EnableMock()` returns a `*mockCalculator` on which callers set `OnCalculate(err)` expectations. `DisableMock(t)` asserts all expectations were consumed. | Mock `Calculate` still calls `invoice.MergeValidationIssues` to mirror the real pipeline — this is intentional. `AssertExpectations` only fires if the option was Set (mo.Some); un-set options are not checked. |
| `draftuntil.go` | Sets `DraftUntil` from `max(CollectionAt, CreatedAt) + DraftPeriod`. Must run after `StandardInvoiceCollectionAt`. | When `AutoAdvance=false`, sets `DraftUntil = nil` unconditionally. |
| `dueat.go` | Sets `DueAt` relative to `DraftUntil` (auto-advance) or `IssuedAt` (manual). Must run after `CalculateDraftUntil`. Truncates to second precision for Stripe compatibility. | Returns nil (no error) when prerequisite fields are absent rather than returning an error — callers must not rely on DueAt being set after a single pass. |
| `gatheringrealtime.go` | Used only in the `GatheringInvoiceWithLiveData` pass to backfill synthetic IDs and timestamps on in-memory DetailedLines that were never persisted to the DB. | Only generates IDs/timestamps when fields are zero-valued, so it is safe to call multiple times without stomping existing values. |

## Anti-Patterns

- Adding DB queries inside a calculation function — all external data must be injected via StandardInvoiceCalculatorDependencies or GatheringInvoiceCalculatorDependencies before the pipeline runs.
- Adding a calculation step that is not registered in InvoiceCalculations — unregistered steps are never called by the Calculator.
- Modifying the `InvoiceCalculations` slice at runtime (e.g. in init() or constructors) — the registry is a package-level var meant to be read-only after package init.
- Directly calling individual step functions (e.g. `CalculateDraftUntil(inv)`) from outside the package instead of going through `Calculator.Calculate` — bypasses ordering guarantees and error aggregation.
- Returning a nil error from a gathering calculation when lines are not expanded — all calculations must validate preconditions and return explicit errors so callers know the output is invalid.

## Decisions

- **Stateless function-per-concern design with a central ordered registry** — Invoice calculations have ordering dependencies (collection → draft → due) but are otherwise independent concerns. A flat ordered slice makes the sequence explicit and testable in isolation; each function can be unit-tested without constructing the full Calculator.
- **Dependencies injected as structs rather than interfaces per function** — StandardInvoiceCalculatorDependencies groups all external inputs (FeatureMeters, RatingService, TaxCodes, LineEngines) that calculation steps may need. This avoids threading many parameters through every function signature and makes it cheap to add a new dependency without changing every step signature.
- **Separate calculation passes for GatheringInvoice vs StandardInvoice** — GatheringInvoice only needs scheduling fields (NextCollectionAt, ServicePeriod, discount IDs); it does not carry ValidationIssues. StandardInvoice needs the full set including detailed-line recalculation and tax snapshotting. The three-slice registry (Invoice, GatheringInvoice, GatheringInvoiceWithLiveData) makes these distinct without conditional branching inside step functions.

## Example: Add a new StandardInvoice calculation step that requires no external dependencies

```
// In a new file, e.g. myfield.go:
package invoicecalc

import "github.com/openmeterio/openmeter/openmeter/billing"

func CalculateMyField(i *billing.StandardInvoice) error {
    // Guard: lines must be present if you iterate them
    if !i.Lines.IsPresent() {
        return errors.New("lines must be expanded")
    }
    // ... compute and assign i.MyField
    return nil
}

// In calculator.go, append to InvoiceCalculations.Invoice:
// ...
```

<!-- archie:ai-end -->
