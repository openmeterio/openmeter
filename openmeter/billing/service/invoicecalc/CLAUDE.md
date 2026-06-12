# invoicecalc

<!-- archie:ai-start -->

> Pure, stateless calculation pipeline for billing invoices. Each calculation is a free function that mutates a *billing.StandardInvoice or *billing.GatheringInvoice in place; the Calculator interface runs ordered lists of them to derive collection/draft/due dates, recalculate detailed lines and totals, stamp tax codes, and resolve service periods.

## Patterns

**Calculations are free functions, registered in an ordered list** — Every calculation is a package-level func that takes an invoice pointer (and optionally a deps struct) and returns error. New calculations must be added to the appropriate slice in the InvoiceCalculations var (Invoice, GatheringInvoice, or GatheringInvoiceWithLiveData) — order matters because later steps depend on earlier results (e.g. CalculateDueAt reads DraftUntil set by CalculateDraftUntil). (`func CalculateDueAt(i *billing.StandardInvoice) error { ... } registered via WithNoDependencies(CalculateDueAt) in InvoiceCalculations.Invoice`)
**Dependency adapter wrappers** — Calculations that need no deps are wrapped with WithNoDependencies (standard) or WithNoGatheringDependencies (gathering) to adapt a func(inv) error into the StandardInvoiceCalculation / GatheringInvoiceCalculation signature. Calculations needing rating/tax/line-engine data take StandardInvoiceCalculatorDependencies directly. (`WithNoDependencies(CalculateDraftUntil)`)
**In-place mutation, never construct new invoices** — Calculations mutate the passed invoice pointer's fields (i.CollectionAt, i.DueAt, invoice.Totals, invoice.Period, line.TaxConfig) rather than returning a new value. Lines are read via invoice.Lines.OrEmpty()/IsPresent()/IsAbsent() and must be expanded — most calculations return errors.New("lines must be expanded") when they are not. (`i.CollectionAt = lo.ToPtr(collectionAt)`)
**Errors are joined, not short-circuited; merged into ValidationIssues** — calculator.applyCalculations runs ALL standard calculations and errors.Join-s their errors, then folds the joined error into the invoice via invoice.MergeValidationIssues(billing.ValidationWithComponent(billing.ValidationComponentOpenMeter, outErr), ...). Gathering invoices have no ValidationIssues so CalculateGatheringInvoice just returns errors.Join(errs...). (`return invoice.MergeValidationIssues(billing.ValidationWithComponent(billing.ValidationComponentOpenMeter, outErr), billing.ValidationComponentOpenMeter)`)
**Deleted lines are always skipped** — Every line-iterating calculation filters out lines where DeletedAt != nil (or line.IsDeleted()) before contributing to collection-at, totals, or period. Deleted lines contribute totals.Totals{} to the sum. (`if line.DeletedAt != nil { return totals.Totals{} }`)
**Line-engine delegation for detailed-line recalculation** — RecalculateDetailedLinesAndTotals groups lines by line.Engine, resolves each via deps.LineEngines.Get(type), and only recalculates engines that implement billing.LineCalculator (type-assert, skip otherwise). It validates CalculateLinesInput, the output, and billing.ValidateStandardLineIDsMatchExactly before invoice.Lines.ReplaceLinesByID, then re-sums totals. (`lineCalculator, ok := lineEngine.(billing.LineCalculator); if !ok { continue }`)
**Tax codes pre-resolved into a map, never queried here** — TaxCodes is a map[string]taxcode.TaxCode keyed by Stripe code, built by the caller before the calculator runs so SnapshotTaxConfigIntoLines stamps TaxCodeID + entity in one pass without DB access. The package does no I/O. (`tc, ok := deps.TaxCodes.Get(line.TaxConfig.Stripe.Code)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `calculator.go` | Defines the Calculator interface, the StandardInvoiceCalculation/GatheringInvoiceCalculation func types, the InvoiceCalculations registry of ordered pipelines, the deps structs, LineEngineResolver, TaxCodes map, and the WithNoDependencies/WithNoGatheringDependencies adapters. New(): Calculator returns the real impl. | Adding a calculation requires registering it in the correct InvoiceCalculations slice and in the right order; GatheringInvoiceWithLiveData runs against a *StandardInvoice but must be a gathering-status invoice (guarded by a status check). |
| `collectionat.go` | GatheringInvoiceCollectionAt (earliest InvoiceAt, anchored vs subscription alignment via timeutil recurrence) and StandardInvoiceCollectionAt (latest InvoiceAt of non-deleted metered lines + collection interval, honoring per-line OverrideCollectionPeriodEnd). | StandardInvoiceCollectionAt only considers lines where line.DependsOnMeteredQuantity() — flat-fee lines are ignored. Returns nil CollectionAt when no metered lines. |
| `details.go` | RecalculateDetailedLinesAndTotals (line-engine delegation + totals.Sum), newDetailedLines (maps rating.DetailedLine -> billing.DetailedLine, defaulting Category to stddetailedline.CategoryRegular and PaymentTerm to InArrears), and MergeGeneratedDetailedLines (persists rating.GenerateDetailedLinesResult onto a parent line with ID reuse). | Each generated/mapped detailed line is Validate()-d; ReplaceLinesByID requires IDs to match exactly (ValidateStandardLineIDsMatchExactly). MergeGeneratedDetailedLines assigns sequential Index via DetailedLinesWithIDReuse. |
| `taxconfig.go` | SnapshotTaxConfigIntoLines merges invoice DefaultTaxConfig into each line (billing.MergeTaxConfigs(billing.FromProductCatalog(...), line.TaxConfig)) and stamps resolved TaxCodeID + TaxCode entity from deps.TaxCodes. | No-op for gathering invoices (StandardInvoiceStatusGathering). Line stripe code wins over DefaultTaxConfig. Existing TaxCodeID is preserved; only TaxCode entity is always (re)stamped. billing.TaxConfig.Equal is ID-only for TaxCode but DOES detect nil-vs-stamped so the adapter re-upserts. |
| `draftuntil.go` | CalculateDraftUntil — DraftUntil = max(CollectionAt, CreatedAt) + DraftPeriod, only when Invoicing.AutoAdvance is true (else nil). | Nils DraftUntil when AutoAdvance is off; CalculateDueAt depends on DraftUntil so ordering matters. |
| `dueat.go` | CalculateDueAt — auto-advance path adds DueAfter to DraftUntil; manual path adds DueAfter to IssuedAt. Result truncated to seconds (Stripe precision). | Returns early (leaves DueAt unset) when DraftUntil is nil (auto) or IssuedAt is nil (manual). |
| `period.go` | CalculateStandardInvoiceServicePeriod / CalculateGatheringInvoiceServicePeriod — span min From / max To across non-deleted lines (StandardLine uses line.Period, GatheringLine uses line.ServicePeriod). | Standard sets invoice.Period to a *ClosedPeriod (nil when no lines); gathering sets invoice.ServicePeriod to a value ClosedPeriod. |
| `mock.go` | MockableInvoiceCalculator wraps an upstream Calculator; EnableMock/DisableMock inject errors via OnCalculate/OnCalculateGatheringInvoice(WithLiveData). Used in tests to simulate calculation failures merged as ValidationIssues. | Mock replays the same MergeValidationIssues wrapping as the real Calculate so injected errors surface as ValidationComponentOpenMeter issues; AssertExpectations fails if a set expectation was never called. |

## Anti-Patterns

- Performing DB / network I/O inside a calculation — all data (rating, tax codes, line engines) must arrive via the deps struct; the package is pure and stateless.
- Returning early on the first error in the standard pipeline — applyCalculations joins all errors and folds them into the invoice's ValidationIssues; do not short-circuit.
- Including deleted lines (DeletedAt != nil) in collection-at, totals, or period computations.
- Adding a calculation without registering it in the correct InvoiceCalculations slice, or registering in an order that breaks a downstream dependency (e.g. DueAt before DraftUntil).
- Constructing a new invoice/line instead of mutating the passed pointer, or reading lines without checking IsPresent()/IsAbsent() (operating on unexpanded lines).

## Decisions

- **Calculations are independent free functions registered in ordered slices rather than methods on a stateful service.** — Keeps each step unit-testable in isolation (see collectionat_test.go, taxconfig_test.go) and makes the full pipeline composition explicit and reorderable in calculator.go.
- **Tax codes and feature meters are pre-resolved into maps passed via StandardInvoiceCalculatorDependencies.** — Lets the calculator stay pure/side-effect-free and stamp all lines in a single in-memory pass without per-line DB lookups; the caller batches resolution upfront.
- **Three distinct pipelines (Invoice, GatheringInvoice, GatheringInvoiceWithLiveData) instead of one.** — Gathering invoices have no ValidationIssues and a different line model (GatheringLine), while live-data gathering reuses standard-invoice calculations to populate detailed line previews — each needs a tailored ordered set.

## Example: Adding a new standard-invoice calculation that mutates the invoice and registering it in the pipeline

```
package invoicecalc

import (
	"github.com/samber/lo"
	"github.com/openmeterio/openmeter/openmeter/billing"
)

// CalculateDraftUntil mutates the invoice in place and returns an error only on real failure.
func CalculateDraftUntil(i *billing.StandardInvoice) error {
	if !i.Workflow.Config.Invoicing.AutoAdvance {
		i.DraftUntil = nil
		return nil
	}
	collectionAt := lo.Latest(lo.FromPtrOr(i.CollectionAt, i.CreatedAt), i.CreatedAt)
	draftUntil, _ := i.Workflow.Config.Invoicing.DraftPeriod.AddTo(collectionAt)
// ...
```

<!-- archie:ai-end -->
