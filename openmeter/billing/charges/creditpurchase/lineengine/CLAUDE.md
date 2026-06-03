# lineengine

<!-- archie:ai-start -->

> Implements billing.LineEngine for the credit-purchase line type (LineEngineTypeChargeCreditPurchase). Delegates line amount calculation to rating.Service.GenerateDetailedLines and explicitly disallows progressive billing via SplitGatheringLine.

## Patterns

**Full billing.LineEngine interface compliance** — Implement all billing.LineEngine methods; unused lifecycle hooks return input.Lines unchanged or nil error. Compile-time assertions enforce the contract. (`var _ billing.LineEngine = (*Engine)(nil); var _ billing.LineCalculator = (*Engine)(nil)`)
**Config.Validate() before construction** — New(Config) validates RatingService non-nil before constructing the Engine. (`func New(config Config) (*Engine, error) { if err := config.Validate(); err != nil { return nil, err }; return &Engine{ratingService: config.RatingService}, nil }`)
**CalculateLines delegates to rating + merge** — Line calc calls ratingService.GenerateDetailedLines per standard line then merges via invoicecalc.MergeGeneratedDetailedLines — never computes amounts inline. (`generated, _ := e.ratingService.GenerateDetailedLines(stdLine)
invoicecalc.MergeGeneratedDetailedLines(stdLine, generated)`)
**SplitGatheringLine always errors** — Credit purchases are never progressively billed; SplitGatheringLine always returns an error — do not add partial-period splitting. (`func (e *Engine) SplitGatheringLine(_ context.Context, _ billing.SplitGatheringLineInput) (billing.SplitGatheringLineResult, error) { return billing.SplitGatheringLineResult{}, fmt.Errorf("credit purchase line is not progressively billed") }`)
**ChargeID required on every line** — CalculateLines validates stdLine.ChargeID is non-nil for every line before delegating, erroring immediately if missing. (`if stdLine.ChargeID == nil { return nil, fmt.Errorf("credit purchase standard line[%s]: charge id is required", stdLine.ID) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `engine.go` | Entire Engine struct with Config, New, all billing.LineEngine method stubs, and CalculateLines. | ChargeID must be non-nil on every StandardLine; call invoicecalc.MergeGeneratedDetailedLines after GenerateDetailedLines or detailed lines stay unattached. |

## Anti-Patterns

- Implementing SplitGatheringLine with real partial-period logic — credit purchases are not progressively billed.
- Computing line amounts in the engine instead of delegating to ratingService.GenerateDetailedLines.
- Skipping invoicecalc.MergeGeneratedDetailedLines after GenerateDetailedLines — leaves detailed lines unattached.
- Adding a lifecycle hook doing non-trivial work without checking ChargeID validity first.

## Decisions

- **Engine is a thin dispatcher to rating.Service rather than computing amounts itself.** — rating.Service owns all billing-period and amount calculation; the engine only orchestrates and complies with lifecycle hooks.

## Example: CalculateLines delegating to the rating service

```
func (e *Engine) CalculateLines(input billing.CalculateLinesInput) (billing.StandardLines, error) {
	for _, stdLine := range input.Lines {
		if stdLine.ChargeID == nil { return nil, fmt.Errorf("credit purchase standard line[%s]: charge id is required", stdLine.ID) }
		generated, err := e.ratingService.GenerateDetailedLines(stdLine)
		if err != nil { return nil, fmt.Errorf("generating detailed lines for line[%s]: %w", stdLine.ID, err) }
		if err := invoicecalc.MergeGeneratedDetailedLines(stdLine, generated); err != nil { return nil, err }
	}
	return input.Lines, nil
}
```

<!-- archie:ai-end -->
