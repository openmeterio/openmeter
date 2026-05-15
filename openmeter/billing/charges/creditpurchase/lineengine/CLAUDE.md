# lineengine

<!-- archie:ai-start -->

> Implements billing.LineEngine for the credit-purchase line type (LineEngineTypeChargeCreditPurchase). Delegates line amount calculation to rating.Service.GenerateDetailedLines and explicitly disallows progressive billing via SplitGatheringLine.

## Patterns

**Full billing.LineEngine interface compliance** — Engine must implement all billing.LineEngine methods: GetLineEngineType, IsLineBillableAsOf, SplitGatheringLine, BuildStandardInvoiceLines, OnCollectionCompleted, OnStandardInvoiceCreated, OnMutableStandardLinesDeleted, OnUnsupportedCreditNote, OnInvoiceIssued, OnPaymentAuthorized, OnPaymentSettled. Unused lifecycle hooks return input.Lines unchanged or nil error. (`var _ billing.LineEngine = (*Engine)(nil); var _ billing.LineCalculator = (*Engine)(nil)`)
**Config.Validate() before construction** — New(Config) validates that RatingService is non-nil before constructing the Engine. (`func New(config Config) (*Engine, error) { if err := config.Validate(); err != nil { return nil, err }; return &Engine{ratingService: config.RatingService}, nil }`)
**CalculateLines delegates to ratingService.GenerateDetailedLines + invoicecalc.MergeGeneratedDetailedLines** — Line calculation calls ratingService.GenerateDetailedLines for each standard line then merges the result via invoicecalc.MergeGeneratedDetailedLines — never computes amounts inline. (`generatedDetailedLines, err := e.ratingService.GenerateDetailedLines(stdLine)
if err := invoicecalc.MergeGeneratedDetailedLines(stdLine, generatedDetailedLines); err != nil { ... }`)
**SplitGatheringLine always errors** — Credit purchases are never progressively billed; SplitGatheringLine must always return an error — do not add partial-period splitting logic. (`func (e *Engine) SplitGatheringLine(_ context.Context, _ billing.SplitGatheringLineInput) (billing.SplitGatheringLineResult, error) { return billing.SplitGatheringLineResult{}, fmt.Errorf("credit purchase line is not progressively billed") }`)
**ChargeID required on every line in CalculateLines** — CalculateLines validates that stdLine.ChargeID is non-nil for every line before delegating to ratingService — return an error immediately if missing. (`if stdLine.ChargeID == nil { return nil, fmt.Errorf("credit purchase standard line[%s]: charge id is required", stdLine.ID) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `engine.go` | Single file containing the entire Engine struct with Config, New constructor, all billing.LineEngine method stubs, and CalculateLines. | ChargeID must be non-nil on every StandardLine; invoicecalc.MergeGeneratedDetailedLines must be called after GenerateDetailedLines — omitting it leaves detailed lines unattached. |

## Anti-Patterns

- Implementing SplitGatheringLine with real partial-period logic — credit purchases are not progressively billed.
- Computing line amounts directly in the engine instead of delegating to ratingService.GenerateDetailedLines.
- Skipping invoicecalc.MergeGeneratedDetailedLines after GenerateDetailedLines — leaves detailed lines unattached to the standard line.
- Adding a new lifecycle hook method that does non-trivial work without checking ChargeID validity first.

## Decisions

- **Engine is a thin dispatcher to rating.Service rather than computing amounts itself.** — rating.Service owns all billing-period and amount calculation; the engine's role is orchestration and lifecycle hook compliance only.

## Example: Implementing a new lifecycle hook that delegates to rating service

```
import (
	"context"
	"fmt"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/service/invoicecalc"
)

func (e *Engine) CalculateLines(input billing.CalculateLinesInput) (billing.StandardLines, error) {
	for _, stdLine := range input.Lines {
		if stdLine.ChargeID == nil {
			return nil, fmt.Errorf("credit purchase standard line[%s]: charge id is required", stdLine.ID)
		}
		generated, err := e.ratingService.GenerateDetailedLines(stdLine)
		if err != nil {
			return nil, fmt.Errorf("generating detailed lines for line[%s]: %w", stdLine.ID, err)
// ...
```

<!-- archie:ai-end -->
