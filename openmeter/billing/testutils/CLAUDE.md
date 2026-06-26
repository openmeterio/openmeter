# testutils

<!-- archie:ai-start -->

> Shared test doubles for the billing domain. Currently provides NoopLineEngine, a no-op billing.LineEngine for embedding in line-engine fakes; it deliberately does NOT implement billing.LineCalculator.

## Patterns

**Noop engine implementing every hook** — NoopLineEngine satisfies var _ billing.LineEngine and returns pass-through results (input.Lines, nil) for every lifecycle hook; Build* methods just call input.GatheringLines.ToStandardLines(input.Invoice.ID). (`func (NoopLineEngine) OnCollectionCompleted(_ context.Context, input billing.OnCollectionCompletedInput) (billing.StandardLines, error) { return input.Lines, nil }`)
**Required EngineType via panic** — GetLineEngineType panics when EngineType is the empty string — tests must set NoopLineEngine{EngineType: ...}. Panic is acceptable here because it is test-only code. (`func (e NoopLineEngine) GetLineEngineType() billing.LineEngineType { if e.EngineType == "" { panic("engine type is required") }; return e.EngineType }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `lineengine.go` | NoopLineEngine struct + all billing.LineEngine hook implementations | Intentionally omits LineCalculator (BuildStandardInvoiceLines exists but CalculateLines does not); embed it and add the calculator separately when a fake needs calculation |

## Anti-Patterns

- Importing application wiring (app/common) into test helpers here — keep testutils dependent only on openmeter/billing
- Assuming NoopLineEngine implements billing.LineCalculator
- Using NoopLineEngine without setting EngineType (panics on GetLineEngineType)

## Decisions

- **Noop engine is a struct with an exported EngineType field rather than a constructor** — Lets tests pick the engine type inline and embed the noop in larger fakes that override only specific hooks

<!-- archie:ai-end -->
