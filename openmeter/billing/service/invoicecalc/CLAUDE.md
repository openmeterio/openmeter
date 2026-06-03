# invoicecalc

<!-- archie:ai-start -->

> Pure, stateless invoice calculation pipeline that applies ordered transformation functions to StandardInvoice and GatheringInvoice to compute derived fields (collection/draft/due dates, service periods, discount IDs, detailed line totals, tax config snapshots) without touching the DB. All external data must be pre-resolved and injected via dependency structs before the pipeline runs.

## Patterns

**Typed calculation function signatures** — Every step matches StandardInvoiceCalculation = func(*billing.StandardInvoice, StandardInvoiceCalculatorDependencies) error or its GatheringInvoice counterpart. New steps must match one of these two signatures. (`func MyCalc(inv *billing.StandardInvoice, deps StandardInvoiceCalculatorDependencies) error { ... }`)
**Register steps in InvoiceCalculations registry** — Active steps are registered in the package-level InvoiceCalculations var in calculator.go under Invoice, GatheringInvoice, or GatheringInvoiceWithLiveData; steps run in declaration order and ordering matters (CollectionAt before DraftUntil before DueAt). (`InvoiceCalculations.Invoice = []StandardInvoiceCalculation{ WithNoDependencies(StandardInvoiceCollectionAt), WithNoDependencies(CalculateDraftUntil), RecalculateDetailedLinesAndTotals, SnapshotTaxConfigIntoLines }`)
**WithNoDependencies / WithNoGatheringDependencies wrappers** — Wrap a func(*billing.StandardInvoice) error into a StandardInvoiceCalculation when the step needs no deps; do not accept and ignore the deps parameter — wrap instead. (`WithNoDependencies(CalculateDraftUntil)`)
**Error aggregation via MergeValidationIssues** — applyCalculations joins step errors with errors.Join and funnels StandardInvoice errors through invoice.MergeValidationIssues(billing.ValidationWithComponent(...)). Steps return plain errors; GatheringInvoice returns raw joined errors (no ValidationIssues field). (`return invoice.MergeValidationIssues(billing.ValidationWithComponent(billing.ValidationComponentOpenMeter, outErr), billing.ValidationComponentOpenMeter)`)
**Guard Lines.IsPresent() before iterating** — Every step iterating lines must guard with i.Lines.IsPresent() and return an explicit error when absent — never silently skip an absent Lines optional. (`if !i.Lines.IsPresent() { return errors.New("lines must be expanded") }`)
**LineEngine dispatch via LineEngineResolver** — RecalculateDetailedLinesAndTotals groups lines by line.Engine and calls deps.LineEngines.Get(lineEngineType); engines not implementing billing.LineCalculator are skipped. ValidateStandardLineIDsMatchExactly enforces engines return exactly the line IDs they received. (`lineEngine, err := deps.LineEngines.Get(lineEngineType); updatedLines, err := lineEngine.CalculateLines(input)`)
**Pre-resolved TaxCodes map** — SnapshotTaxConfigIntoLines reads from deps.TaxCodes (map[string]taxcode.TaxCode keyed by Stripe code) pre-populated before the calculator runs; tax steps must read this map, not query the DB. (`tc, ok := deps.TaxCodes.Get(line.TaxConfig.Stripe.Code)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `calculator.go` | Calculator interface, invoiceCalculatorsByType registry, StandardInvoiceCalculatorDependencies, GatheringInvoiceCalculatorDependencies, LineEngineResolver, TaxCodes, New(); applyCalculations runs all steps and merges errors. | Step registration order in InvoiceCalculations matters — CollectionAt must precede DraftUntil which must precede DueAt; adding a step without checking ordering silently produces wrong derived fields. |
| `details.go` | RecalculateDetailedLinesAndTotals (engine dispatch + totals rollup) and MergeGeneratedDetailedLines (materialises rating.GenerateDetailedLinesResult into a parent StandardLine). | ValidateStandardLineIDsMatchExactly hard-errors if an engine creates or drops lines — engines must return exactly the IDs they received. |
| `collectionat.go` | Computes CollectionAt for StandardInvoice and NextCollectionAt for GatheringInvoice using timeutil.NewRecurrenceFromISODuration for subscription/anchored configs. | Only metered lines (DependsOnMeteredQuantity()) contribute to StandardInvoice collection time; flat-fee and deleted lines are excluded. OverrideCollectionPeriodEnd overrides the default interval calc for a line. |
| `taxconfig.go` | Merges Workflow.Config.Invoicing.DefaultTaxConfig into each line via productcatalog.MergeTaxConfigs, then stamps TaxCode entity and TaxCodeID from deps.TaxCodes. | No-op for gathering invoices (returns nil immediately). Does not overwrite an existing TaxCodeID but always stamps the TaxCode entity if found. |
| `draftuntil.go` | Sets DraftUntil = max(CollectionAt, CreatedAt) + DraftPeriod; must run after StandardInvoiceCollectionAt. | When AutoAdvance=false, sets DraftUntil = nil unconditionally. |
| `dueat.go` | Sets DueAt relative to DraftUntil (auto-advance) or IssuedAt (manual); truncates to second precision for Stripe. | Returns nil (no error) when prerequisite fields are absent — callers must not assume DueAt is set after a single pass. |
| `gatheringrealtime.go` | GatheringInvoiceWithLiveData pass: backfills synthetic IDs/timestamps on in-memory DetailedLines never persisted to the DB. | Only fills zero-valued fields, so it is safe to call multiple times without stomping existing values. |
| `mock.go` | MockableInvoiceCalculator wrapping real Calculator + optional mockCalculator; EnableMock() returns a *mockCalculator with OnCalculate(err) expectations, DisableMock(t) asserts consumption. | Mock Calculate still calls invoice.MergeValidationIssues to mirror the real pipeline; AssertExpectations only fires when the option was Set (mo.Some). |

## Anti-Patterns

- Adding DB queries inside a calculation function — all external data must be injected via the dependency structs.
- Adding a calculation step not registered in InvoiceCalculations — unregistered steps are never called.
- Mutating the InvoiceCalculations slice at runtime (init/constructors) — the registry is read-only after package init.
- Calling individual step functions directly from outside the package instead of Calculator.Calculate — bypasses ordering and error aggregation.
- Returning nil error when lines are not expanded — steps must validate preconditions and return explicit errors.

## Decisions

- **Stateless function-per-concern design with a central ordered registry.** — Calculations have ordering dependencies (collection→draft→due) but are otherwise independent; a flat ordered slice makes the sequence explicit and each function unit-testable in isolation.
- **Dependencies injected as grouped structs rather than per-function interfaces.** — Groups all external inputs (FeatureMeters, RatingService, TaxCodes, LineEngines) so adding a dependency does not change every step signature.
- **Separate calculation passes for GatheringInvoice vs StandardInvoice (three-slice registry).** — GatheringInvoice needs only scheduling fields and carries no ValidationIssues; StandardInvoice needs detailed-line recalculation and tax snapshotting — distinct passes avoid conditional branching inside steps.

## Example: Adding a StandardInvoice calculation step with no external dependencies

```
// myfield.go
package invoicecalc

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

func CalculateMyField(i *billing.StandardInvoice) error {
	if !i.Lines.IsPresent() {
		return errors.New("lines must be expanded")
	}
	// ... compute and assign i.MyField
	return nil
// ...
```

<!-- archie:ai-end -->
