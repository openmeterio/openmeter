# testutils

<!-- archie:ai-start -->

> Test utilities for the billing domain — provides NoopLineEngine, a billing.LineEngine stub for embedding in line-engine fakes. Intentionally does not implement billing.LineCalculator so test doubles selectively override only the hooks they need.

## Patterns

**NoopLineEngine for test doubles** — Pass-through billing.LineEngine (returns inputs unchanged, IsLineBillableAsOf returns true); embed and override only relevant methods. (`type MyTestEngine struct { testutils.NoopLineEngine }; func (e *MyTestEngine) OnCollectionCompleted(ctx, input) (billing.StandardLines, error) { /* custom */ }`)
**EngineType field must be set** — GetLineEngineType() panics if EngineType is empty; always initialise. (`engine := testutils.NoopLineEngine{EngineType: billing.LineEngineTypeInvoice}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `lineengine.go` | Defines NoopLineEngine — implements billing.LineEngine but not billing.LineCalculator; all hooks pass-through. | Do not run BuildStandardInvoiceLines in production via this type — it contains test-only stub logic (ToStandardLines) that bypasses snapshotting. |

## Anti-Patterns

- Using NoopLineEngine in production wiring — it skips quantity snapshotting and rating.
- Importing app/common from testutils — build deps from underlying constructors to avoid import cycles.
- Adding production utility/domain code here — testutils must only contain test helpers.

## Decisions

- **NoopLineEngine intentionally does not implement billing.LineCalculator.** — Separates the minimum LineEngine registration interface from the optional calculator extension; doubles implement only what they need.

## Example: Embedding NoopLineEngine to override one hook in a test

```
import "github.com/openmeterio/openmeter/openmeter/billing/testutils"

type captureEngine struct {
    testutils.NoopLineEngine
    captured billing.StandardLines
}
func (e *captureEngine) OnCollectionCompleted(_ context.Context, input billing.OnCollectionCompletedInput) (billing.StandardLines, error) {
    e.captured = input.Lines
    return input.Lines, nil
}
// engine := &captureEngine{NoopLineEngine: testutils.NoopLineEngine{EngineType: billing.LineEngineTypeInvoice}}
```

<!-- archie:ai-end -->
