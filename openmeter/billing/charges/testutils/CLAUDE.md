# testutils

<!-- archie:ai-start -->

> Shared test infrastructure for the charges domain: provides MockHandlers (no-op flatfee.Handler, creditpurchase.Handler, usagebased.Handler with deterministic ledger references) and a NewServices factory that wires the full charges stack from raw dependencies, mirroring production wiring without importing app/common.

## Patterns

**MockHandlers as zero-side-effect handler implementations** — mockFlatFeeHandler, mockCreditPurchaseHandler, mockUsageBasedHandler implement all handler methods. Ledger callbacks return a fresh ULID-based ledgertransaction.GroupReference per invocation so tests can distinguish which callback ran. (`handlers := testutils.NewMockHandlers()
flatFeeService, _ := flatfeeservice.New(flatfeeservice.Config{Handler: handlers.FlatFee, ...})`)
**NewServices wires the full charges stack** — NewServices(t, Config) constructs meta adapter, locker, lineage adapter/service, each charge-type adapter/service, line engines, and the top-level charges.Service in one call; also calls BillingService.RegisterLineEngine for flatfee and usagebased. (`svcs, err := testutils.NewServices(t, testutils.Config{ Client: dbClient, BillingService: billingSvc, FlatFeeHandler: handlers.FlatFee, ... })`)
**Config.Validate() guards all required dependencies** — Config.Validate() returns joined errors for any nil required field (Client, BillingService, FeatureService, StreamingConnector, all three handlers). RecognizerService defaults to recognizer.NoopService{} if nil. (`if config.RecognizerService == nil { config.RecognizerService = recognizer.NoopService{} }
if err := config.Validate(); err != nil { return nil, err }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handlers.go` | Mock implementations of flatfee.Handler, creditpurchase.Handler, usagebased.Handler with compile-time interface assertions and newMockLedgerTransactionGroupReference() returning fresh ULIDs. | When a new method is added to any Handler interface, add a corresponding no-op here or the mock stops compiling. Each callback returns a fresh ULID — tests verifying specific IDs must capture the return value. |
| `service.go` | Config struct, Validate(), Services struct, and NewServices factory constructing the full charges sub-service tree for tests. | NewServices registers flatfee, usagebased, and creditpurchase line engines on BillingService. Adding a new charge type requires adding its adapter, service, and line engine registration here. |

## Anti-Patterns

- Importing app/common from testutils — creates import cycles; build all dependencies from package constructors directly
- Adding stateful side effects to mock handlers without a Reset() method — causes test pollution across suite cases
- Using NewServices in production wiring (app/common) — this factory is test-only with mock handlers and no real ledger integration
- Relying on a fixed ledger transaction group ID — mock callbacks return fresh ULIDs each invocation

## Decisions

- **MockHandlers returns a new ULID on every ledger callback instead of a fixed constant** — Tests verifying ledger transaction group IDs need unique, distinguishable IDs per callback; a fixed constant would prevent differentiating which callback ran or how many times.
- **NewServices wires all sub-services rather than requiring callers to compose them** — The charges stack has a deep dependency tree; centralizing setup reduces test boilerplate and ensures consistent wiring across integration tests.

## Example: Setting up the full charges stack in an integration test

```
handlers := testutils.NewMockHandlers()
svcs, err := testutils.NewServices(t, testutils.Config{
    Client:                dbClient,
    BillingService:        billingSvc,
    FeatureService:        featureSvc,
    StreamingConnector:    mockStreamingConnector,
    FlatFeeHandler:        handlers.FlatFee,
    CreditPurchaseHandler: handlers.CreditPurchase,
    UsageBasedHandler:     handlers.UsageBased,
})
require.NoError(t, err)
// svcs.ChargesService, svcs.FlatFeeService, svcs.UsageBasedService, svcs.CreditPurchaseService
```

<!-- archie:ai-end -->
