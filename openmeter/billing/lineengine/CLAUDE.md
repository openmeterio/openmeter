# lineengine

<!-- archie:ai-start -->

> Implements billing.LineEngine for the standard invoice line type — quantity snapshotting, split-line-group creation, period truncation, and per-line rating via rating.Service. Registered as LineEngineTypeInvoice in the billing.Service engine registry via app/common/charges.go.

## Patterns

**Config struct with Validate() + New()** — Engine constructed via Config.Validate() + New(Config) returning (*Engine, error); all deps injected through Config, never construct Engine directly. (`func New(config Config) (*Engine, error) { if err := config.Validate(); err != nil { return nil, err }; return &Engine{...}, nil }`)
**Narrow SplitLineGroupAdapter interface** — splitlinegroup.go defines SplitLineGroupAdapter with only CreateSplitLineGroup/GetSplitLineGroupHeaders; never import full billing.Adapter or ent/db (prevents cycles). (`type SplitLineGroupAdapter interface { CreateSplitLineGroup(ctx, billing.CreateSplitLineGroupAdapterInput) (billing.SplitLineGroup, error); GetSplitLineGroupHeaders(...) }`)
**QuantitySnapshotter interface for dependency inversion** — stdinvoice.go defines QuantitySnapshotter; concrete impl is billingservice.Service. Engine never calls the service directly. (`type QuantitySnapshotter interface { SnapshotLineQuantities(ctx context.Context, invoice billing.StandardInvoice, lines billing.StandardLines) error }`)
**billing.ValidationIssue for snapshot errors** — When SnapshotLineQuantities returns ErrSnapshotInvalidDatabaseState, convert to billing.ValidationIssue (Critical) so callers can type-assert. (`if _, ok := lo.ErrorsAs[*billing.ErrSnapshotInvalidDatabaseState](err); ok { return nil, billing.ValidationIssue{Severity: billing.ValidationIssueSeverityCritical, ...} }`)
**isPeriodEmptyConsideringTruncations** — Usage-based lines check emptiness after truncation to streaming.MinimumWindowSizeDuration; flat-fee lines always return false. (`return line.GetServicePeriod().Truncate(streaming.MinimumWindowSizeDuration).IsEmpty(), nil`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `engine.go` | Implements billing.LineEngine: GetLineEngineType, OnCollectionCompleted, OnStandardInvoiceCreated, OnInvoiceIssued, OnPaymentAuthorized/Settled. OnCollectionCompleted is the primary snapshot hook. | OnCollectionCompleted short-circuits if QuantitySnapshotedAt is set at/after default collection time — add no snapshot logic after this guard. |
| `splitlinegroup.go` | SplitGatheringLine splits a gathering line at a time, creates a SplitLineGroup, produces pre/post pairs; ResolveSplitLineGroupHeaders batch-fetches metadata. | ChildUniqueReferenceID is set nil on split lines — the SplitLineGroup owns this reference; never copy from source. |
| `stdinvoice.go` | BuildStandardInvoiceLines converts gathering→standard lines, snapshots, runs CalculateLines (ratingService.GenerateDetailedLines per line). | CalculateLines requires non-empty lines; invoicecalc.MergeGeneratedDetailedLines runs after each GenerateDetailedLines call. |

## Anti-Patterns

- Importing billing.Adapter or ent/db directly — use the narrow SplitLineGroupAdapter interface.
- Calling SnapshotLineQuantities outside OnCollectionCompleted or BuildStandardInvoiceLines — snapshotting is lifecycle-gated.
- Skipping ResolveSplitLineGroupHeaders before CalculateLines on lines with split groups — calculations depend on SplitLineHierarchy.
- Returning plain errors from snapshot paths that should be billing.ValidationIssue.
- Copying ChildUniqueReferenceID from source gathering line to split child lines.

## Decisions

- **Engine registered at Service construction via Service.RegisterLineEngine, not hardcoded.** — Lets charges sub-packages register their own engines without modifying the base service (LineEngine plugin registry pattern).
- **SplitLineGroupAdapter and QuantitySnapshotter are narrow interfaces, not the full billing.Service.** — lineengine is imported by billingservice (the QuantitySnapshotter impl); using the full interface would create a cycle.

## Example: A LineEngine hook converting snapshot errors to ValidationIssue

```
func (e *Engine) OnCollectionCompleted(ctx context.Context, input billing.OnCollectionCompletedInput) (billing.StandardLines, error) {
    if input.Invoice.QuantitySnapshotedAt != nil && !input.Invoice.QuantitySnapshotedAt.Before(input.Invoice.DefaultCollectionAtForStandardInvoice()) {
        return input.Lines, nil
    }
    if err := e.quantitySnapshotter.SnapshotLineQuantities(ctx, input.Invoice, input.Lines); err != nil {
        if _, ok := lo.ErrorsAs[*billing.ErrSnapshotInvalidDatabaseState](err); ok {
            return nil, billing.ValidationIssue{Severity: billing.ValidationIssueSeverityCritical, Code: billing.ErrInvoiceLineSnapshotFailed.Code, Message: err.Error(), Component: billing.ValidationComponentOpenMeterMetering}
        }
        return nil, err
    }
    return input.Lines, nil
}
```

<!-- archie:ai-end -->
