# lineengine

<!-- archie:ai-start -->

> Billing LineEngine/LineCalculator implementation for credit-purchase invoice lines (LineEngineTypeChargeCreditPurchase). Turns gathering lines into rated standard invoice lines and supplies the lifecycle hook surface the billing invoice state machine calls.

## Patterns

**Engine satisfies both billing.LineEngine and billing.LineCalculator** — Compile-time asserted with var _ billing.LineEngine / _ billing.LineCalculator = (*Engine)(nil); GetLineEngineType returns billing.LineEngineTypeChargeCreditPurchase. (`var ( _ billing.LineEngine = (*Engine)(nil); _ billing.LineCalculator = (*Engine)(nil) )`)
**Constructor validates RatingService dependency** — New(Config) returns *Engine; Config.Validate() requires a non-nil rating.Service which is the only dependency. (`func (c Config) Validate() error { if c.RatingService == nil { return fmt.Errorf("rating service is required") } ... }`)
**Credit purchases are never progressively billed** — IsLineBillableAsOf always returns true (no partial-period filtering) and SplitGatheringLine returns an error; encode this invariant rather than implementing splitting. (`func (e *Engine) SplitGatheringLine(...) (...) { return billing.SplitGatheringLineResult{}, fmt.Errorf("credit purchase line is not progressively billed") }`)
**Calculation delegates to rating + invoicecalc merge** — CalculateLines requires Invoice.ID and a non-nil ChargeID per line, calls ratingService.GenerateDetailedLines, then invoicecalc.MergeGeneratedDetailedLines, then stdLine.Validate(). (`generatedDetailedLines, err := e.ratingService.GenerateDetailedLines(stdLine); invoicecalc.MergeGeneratedDetailedLines(stdLine, generatedDetailedLines)`)
**Most lifecycle hooks are inert pass-throughs** — OnCollectionCompleted/OnStandardInvoiceCreated return input.Lines unchanged; OnMutableStandardLinesDeleted/OnInvoiceIssued/OnPaymentAuthorized/OnPaymentSettled/OnUnsupportedCreditNote return nil. Side effects live in the service, not the engine. (`func (e *Engine) OnPaymentSettled(_ context.Context, _ billing.OnPaymentSettledInput) error { return nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `engine.go` | Whole package: Config/New, the engine type tag, line building (BuildStandardInvoiceLines, BuildStandardLinesForGatheringPreview), CalculateLines, and the no-op lifecycle hooks | BuildStandardInvoiceLines converts each gathering line via AsNewStandardLine then routes through CalculateLines; CalculateLines hard-requires stdLine.ChargeID != nil |

## Anti-Patterns

- Implementing SplitGatheringLine or partial-period billing logic - credit purchases bill in full
- Putting credit-grant/payment side effects into engine hooks instead of the creditpurchase service
- Calculating a line whose ChargeID is nil or whose Invoice.ID is empty

## Decisions

- **The engine is calculation-only; ledger/payment side effects stay in the service layer and its hooks are no-ops** — Keeps the billing invoice state machine's engine contract pure while credit/payment realization is driven by creditpurchase.Service
- **Credit-purchase lines are modeled as a flat in-advance price computed at gathering time** — Credit purchases are one-shot, non-progressive charges, so there is no usage to meter or period to split

## Example: Building and rating standard invoice lines for credit purchases

```
func (e *Engine) BuildStandardInvoiceLines(ctx context.Context, input billing.BuildStandardInvoiceLinesInput) (billing.StandardLines, error) {
	stdLines, err := slicesx.MapWithErr(input.GatheringLines, func(gl billing.GatheringLine) (*billing.StandardLine, error) {
		return gl.AsNewStandardLine(input.Invoice.ID)
	})
	if err != nil { return nil, err }
	return e.CalculateLines(billing.CalculateLinesInput{Invoice: input.Invoice, Lines: stdLines})
}
```

<!-- archie:ai-end -->
