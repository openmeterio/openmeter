# lineengine

<!-- archie:ai-start -->

> Implements the billing.LineEngine interface for the standard invoice line type — handles quantity snapshotting, split-line-group creation, period truncation, and per-line rating via rating.Service. Registered as LineEngineTypeInvoice in the billing.Service engine registry via app/common/charges.go.

## Patterns

**Config struct with Validate() + New()** — Engine is constructed via Config.Validate() + New(Config), returning (*Engine, error). All dependencies are injected through Config; never construct Engine directly. (`func New(config Config) (*Engine, error) { if err := config.Validate(); err != nil { return nil, err }; return &Engine{...}, nil }`)
**Narrow SplitLineGroupAdapter interface** — splitlinegroup.go defines SplitLineGroupAdapter with only CreateSplitLineGroup and GetSplitLineGroupHeaders. Engine never imports the full billing.Adapter or openmeter/ent/db — dependency inversion prevents circular imports. (`type SplitLineGroupAdapter interface { CreateSplitLineGroup(ctx, billing.CreateSplitLineGroupAdapterInput) (billing.SplitLineGroup, error); GetSplitLineGroupHeaders(ctx, billing.GetSplitLineGroupHeadersInput) (billing.SplitLineGroupHeaders, error) }`)
**QuantitySnapshotter interface for dependency inversion** — stdinvoice.go defines QuantitySnapshotter with SnapshotLineQuantities; the concrete implementation is billingservice.Service. Engine never calls the service directly. (`type QuantitySnapshotter interface { SnapshotLineQuantities(ctx context.Context, invoice billing.StandardInvoice, lines billing.StandardLines) error }`)
**billing.ValidationIssue for snapshot errors** — When SnapshotLineQuantities returns ErrSnapshotInvalidDatabaseState, OnCollectionCompleted converts it to billing.ValidationIssue with ValidationIssueSeverityCritical — not a plain error — so callers can type-assert. (`if _, isInvalidDatabaseState := lo.ErrorsAs[*billing.ErrSnapshotInvalidDatabaseState](err); isInvalidDatabaseState { return nil, billing.ValidationIssue{Severity: billing.ValidationIssueSeverityCritical, ...} }`)
**isPeriodEmptyConsideringTruncations for period validity** — Usage-based lines check if their period is empty after truncation to streaming.MinimumWindowSizeDuration before billing. Flat-fee lines always return false. (`return line.GetServicePeriod().Truncate(streaming.MinimumWindowSizeDuration).IsEmpty(), nil`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `engine.go` | Implements billing.LineEngine interface: GetLineEngineType, OnCollectionCompleted, OnStandardInvoiceCreated, OnInvoiceIssued, OnPaymentAuthorized, OnPaymentSettled. OnCollectionCompleted is the primary hook for quantity snapshotting. | OnCollectionCompleted short-circuits if QuantitySnapshotedAt is already set and at or after the default collection time — do not add snapshot logic after this guard. |
| `splitlinegroup.go` | SplitGatheringLine splits a gathering line at a point in time, creates a SplitLineGroup if needed, and produces pre/post split line pairs. ResolveSplitLineGroupHeaders batch-fetches group metadata for standard lines. | ChildUniqueReferenceID is set to nil on split lines — the SplitLineGroup owns this reference; never copy it from the source line. |
| `stdinvoice.go` | BuildStandardInvoiceLines converts gathering lines to standard lines, snapshots quantities, and runs CalculateLines. CalculateLines calls ratingService.GenerateDetailedLines per line. | CalculateLines requires lines to be non-empty — guard callers; invoicecalc.MergeGeneratedDetailedLines is called after each GenerateDetailedLines call. |

## Anti-Patterns

- Importing billing.Adapter or openmeter/ent/db directly — use the SplitLineGroupAdapter narrow interface.
- Calling SnapshotLineQuantities from outside OnCollectionCompleted or BuildStandardInvoiceLines — quantity snapshotting is lifecycle-gated.
- Skipping ResolveSplitLineGroupHeaders before CalculateLines on lines that may have split groups — price calculations depend on SplitLineHierarchy being populated.
- Returning plain errors from snapshot paths that should be billing.ValidationIssue — callers use type assertions on ValidationIssue.
- Copying ChildUniqueReferenceID from source gathering line to split child lines.

## Decisions

- **Engine registered at Service construction time via Service.RegisterLineEngine, not hardcoded in billing.Service.** — Allows charges sub-packages (flatfee, usagebased, creditpurchase) to register their own engines without modifying the base service, following the LineEngine plugin registry pattern.
- **SplitLineGroupAdapter and QuantitySnapshotter are narrow interfaces rather than the full billing.Service.** — Prevents circular imports — lineengine is imported by billingservice which is the QuantitySnapshotter implementation; using the full service interface would create a cycle.

## Example: Implementing a new LineEngine hook that converts snapshot errors to ValidationIssue

```
func (e *Engine) OnCollectionCompleted(ctx context.Context, input billing.OnCollectionCompletedInput) (billing.StandardLines, error) {
    // Guard: skip if already snapshotted at or after collection time
    if input.Invoice.QuantitySnapshotedAt != nil &&
        !input.Invoice.QuantitySnapshotedAt.Before(input.Invoice.DefaultCollectionAtForStandardInvoice()) {
        return input.Lines, nil
    }
    if err := e.quantitySnapshotter.SnapshotLineQuantities(ctx, input.Invoice, input.Lines); err != nil {
        if _, isInvalidDatabaseState := lo.ErrorsAs[*billing.ErrSnapshotInvalidDatabaseState](err); isInvalidDatabaseState {
            return nil, billing.ValidationIssue{
                Severity:  billing.ValidationIssueSeverityCritical,
                Code:      billing.ErrInvoiceLineSnapshotFailed.Code,
                Message:   err.Error(),
                Component: billing.ValidationComponentOpenMeterMetering,
            }
        }
// ...
```

<!-- archie:ai-end -->
