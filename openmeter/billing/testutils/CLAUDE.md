# testutils

<!-- archie:ai-start -->

> Test utilities for the billing domain — currently provides NoopLineEngine, a billing.LineEngine stub intended for embedding in line-engine fakes in tests. Intentionally does not implement billing.LineCalculator so test doubles can selectively override only the hooks they need.

## Patterns

**NoopLineEngine for test doubles** — NoopLineEngine implements billing.LineEngine with pass-through no-op behaviour (returns inputs unchanged, IsLineBillableAsOf returns true). Embed it in test fakes to avoid implementing all hooks; only override the methods relevant to the test. (`var _ billing.LineEngine = NoopLineEngine{}; type MyTestEngine struct { testutils.NoopLineEngine }; func (e *MyTestEngine) OnCollectionCompleted(ctx, input) (billing.StandardLines, error) { /* custom logic */ }`)
**EngineType field must be set** — NoopLineEngine.GetLineEngineType() panics if EngineType is empty. Always initialise: NoopLineEngine{EngineType: billing.LineEngineTypeInvoice}. (`engine := testutils.NoopLineEngine{EngineType: billing.LineEngineTypeInvoice}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `lineengine.go` | Defines NoopLineEngine — implements billing.LineEngine but not billing.LineCalculator; all hooks are pass-through. GetLineEngineType panics if EngineType is unset. | Do not call BuildStandardInvoiceLines in production code through this type — it contains test-only stub logic (AsNewStandardLine) that bypasses quantity snapshotting. |

## Anti-Patterns

- Using NoopLineEngine in production wiring — it is a test-only stub that skips quantity snapshotting and rating.
- Importing app/common from testutils — builds test dependencies from underlying package constructors (adapters, services) directly to avoid import cycles.
- Adding production utility code here — testutils must only contain test helpers, never domain logic.

## Decisions

- **NoopLineEngine does not implement billing.LineCalculator intentionally.** — Separates the minimum interface needed for LineEngine registration from the optional calculator extension; test doubles can implement only what their scenario requires.

## Example: Embedding NoopLineEngine in a test-specific line engine that overrides one hook

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

// Usage in test:
engine := &captureEngine{NoopLineEngine: testutils.NoopLineEngine{EngineType: billing.LineEngineTypeInvoice}}
billingService.RegisterLineEngine(engine)
```

<!-- archie:ai-end -->
