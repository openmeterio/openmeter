# lineengine

<!-- archie:ai-start -->

> Concrete line Engine implementing billing.LineEngine + billing.LineCalculator: builds standard invoice lines from gathering lines, splits gathering lines at period boundaries, snapshots quantities, resolves split-line-group hierarchy, and generates/merges detailed rated lines. This is the lifecycle-hook engine the service registers for LineEngineTypeInvoice.

## Patterns

**Config+Validate+New constructor** — Engine is built via New(Config) where Config.Validate() requires non-nil SplitLineGroupAdapter, QuantitySnapshotter, and rating.Service; constructor returns error, never panics. (`func New(config Config) (*Engine, error) { if err := config.Validate(); err != nil { return nil, err }; return &Engine{...}, nil }`)
**Engine satisfies both billing interfaces** — var _ billing.LineEngine and var _ billing.LineCalculator both assert *Engine. Lifecycle hooks (OnCollectionCompleted, OnStandardInvoiceCreated, OnInvoiceIssued, OnPayment*) must keep matching signatures; unused hooks return input.Lines / nil. (`var ( _ billing.LineEngine = (*Engine)(nil); _ billing.LineCalculator = (*Engine)(nil) )`)
**Snapshot gating before quantity capture** — OnCollectionCompleted skips snapshotting when QuantitySnapshotedAt is already past the default collection time, or when collection time is still in the future; only then calls quantitySnapshotter.SnapshotLineQuantities. (`if input.Invoice.QuantitySnapshotedAt != nil && !input.Invoice.QuantitySnapshotedAt.Before(input.Invoice.DefaultCollectionAtForStandardInvoice()) { return input.Lines, nil }`)
**Split-at into pre/post lines via CloneForCreate** — SplitGatheringLine creates a SplitLineGroup if absent, then derives postSplitAtLine via line.CloneForCreate, trims the original to [from, splitAt], and soft-deletes either side whose truncated period is empty (FlatPrice never empty; usage truncated to streaming.MinimumWindowSizeDuration). (`postSplitAtLine, err := line.CloneForCreate(func(l *billing.GatheringLine){ l.ServicePeriod.From = in.SplitAt; l.SplitLineGroupID = lo.ToPtr(splitLineGroupID); l.ChildUniqueReferenceID = nil })`)
**Detailed-line generation then merge+validate** — CalculateLines loops standard lines, calls ratingService.GenerateDetailedLines, then invoicecalc.MergeGeneratedDetailedLines, then stdLine.Validate(); any per-line error is wrapped with the line ID. (`generated, _ := e.ratingService.GenerateDetailedLines(stdLine); invoicecalc.MergeGeneratedDetailedLines(stdLine, generated)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `engine.go` | Config/New, GetLineEngineType (LineEngineTypeInvoice), and lifecycle hooks | Snapshot failures of type *billing.ErrSnapshotInvalidDatabaseState are converted to a critical billing.ValidationIssue, not a raw error |
| `splitlinegroup.go` | SplitLineGroupAdapter interface, SplitGatheringLine, ResolveSplitLineGroupHeaders | splitAt must lie within line.ServicePeriod; empty-after-truncation sides are soft-deleted; isPeriodEmptyConsideringTruncations special-cases FlatPriceType |
| `stdinvoice.go` | QuantitySnapshotter interface + BuildStandardInvoiceLines/CalculateLines/IsLineBillableAsOf | Build path requires non-empty Invoice.ID and GatheringLines; ToStandardLines then ResolveSplitLineGroupHeaders then snapshot, in that order |

## Anti-Patterns

- panic on missing engine type or inputs in production paths — return errors (only the testutils Noop panics)
- Skipping invoicecalc.MergeGeneratedDetailedLines / stdLine.Validate() after generating detailed lines
- Splitting a line at a point outside its ServicePeriod, or reusing ChildUniqueReferenceID on the post-split clone
- Returning a raw error for snapshot DB-state failures instead of a billing.ValidationIssue
- Adding a LineEngine hook to the interface without updating *Engine (and testutils.NoopLineEngine)

## Decisions

- **Engine carries a SplitLineGroupAdapter and QuantitySnapshotter as injected collaborators** — Keeps persistence (split groups) and metering (quantity snapshot) behind narrow interfaces so the engine stays a pure orchestration unit
- **Snapshot DB-state errors become critical ValidationIssues** — Lets the invoice surface a recoverable metering problem to the user instead of failing the whole pipeline opaquely

## Example: Building standard lines: snapshot then rate then validate

```
func (e *Engine) BuildStandardInvoiceLines(ctx context.Context, input billing.BuildStandardInvoiceLinesInput) (billing.StandardLines, error) {
	stdLines, err := e.buildStandardInvoiceLinesWithQuantitySnapshot(ctx, input)
	if err != nil {
		return nil, err
	}
	return e.CalculateLines(billing.CalculateLinesInput{Invoice: input.Invoice, Lines: stdLines})
}
```

<!-- archie:ai-end -->
