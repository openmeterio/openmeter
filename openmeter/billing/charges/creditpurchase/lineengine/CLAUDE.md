# lineengine

<!-- archie:ai-start -->

> Implements billing.LineEngine for the credit-purchase line type (LineEngineTypeChargeCreditPurchase). Delegates detailed line generation to rating.Service; explicitly disallows progressive billing (SplitGatheringLine always errors).

## Patterns

**billing.LineEngine interface compliance** — Engine must implement all billing.LineEngine methods: GetLineEngineType, IsLineBillableAsOf, SplitGatheringLine, BuildStandardInvoiceLines, OnCollectionCompleted, OnStandardInvoiceCreated, OnInvoiceIssued, OnPaymentAuthorized, OnPaymentSettled. Unused lifecycle hooks return input.Lines unchanged or nil error. (`var _ billing.LineEngine = (*Engine)(nil)`)
**Config.Validate() before construction** — New(Config) validates that RatingService is non-nil before constructing the Engine. (`func New(config Config) (*Engine, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**CalculateLines delegates to ratingService.GenerateDetailedLines** — Line calculation calls ratingService.GenerateDetailedLines(stdLine) then invoicecalc.MergeGeneratedDetailedLines for each line; never computes amounts inline. (`generatedDetailedLines, err := e.ratingService.GenerateDetailedLines(stdLine)
if err := invoicecalc.MergeGeneratedDetailedLines(stdLine, generatedDetailedLines); err != nil { ... }`)
**SplitGatheringLine always errors** — Credit purchases are never progressively billed so SplitGatheringLine must return an error — do not add partial-period splitting logic here. (`func (e *Engine) SplitGatheringLine(...) (billing.SplitGatheringLineResult, error) { return billing.SplitGatheringLineResult{}, fmt.Errorf("credit purchase line is not progressively billed") }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `engine.go` | Single file containing the entire Engine implementation; all billing.LineEngine method stubs plus CalculateLines. | ChargeID must be non-nil on every StandardLine passed to CalculateLines — validated before calling ratingService. |

## Anti-Patterns

- Implementing SplitGatheringLine with real logic — credit purchases are not progressively billed.
- Computing line amounts directly in the engine instead of delegating to ratingService.GenerateDetailedLines.
- Skipping invoicecalc.MergeGeneratedDetailedLines after GenerateDetailedLines — leaves detailed lines unattached to the standard line.

## Decisions

- **Engine is a thin dispatcher to rating.Service rather than computing amounts itself.** — rating.Service owns all billing-period and amount calculation; the engine's role is orchestration and lifecycle hook compliance.

<!-- archie:ai-end -->
