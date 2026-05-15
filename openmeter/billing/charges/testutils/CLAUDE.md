# testutils

<!-- archie:ai-start -->

> Shared test infrastructure for the charges domain: provides MockHandlers (no-op implementations of flatfee.Handler, creditpurchase.Handler, usagebased.Handler with deterministic ledger references) and a NewServices factory that wires the full charges stack from raw dependencies for integration tests, mirroring production wiring without importing app/common.

## Patterns

**MockHandlers as zero-side-effect handler implementations** — mockFlatFeeHandler, mockCreditPurchaseHandler, and mockUsageBasedHandler implement all handler interface methods. Ledger callbacks return a fresh ULID-based ledgertransaction.GroupReference per invocation so tests can distinguish which callback ran. (`handlers := testutils.NewMockHandlers()
flatFeeService, _ := flatfeeservice.New(flatfeeservice.Config{Handler: handlers.FlatFee, ...})`)
**NewServices wires the full charges stack** — NewServices(t, Config) constructs meta adapter, locker, lineage adapter/service, each charge-type adapter/service, line engines, and the top-level charges.Service in one call. It also calls BillingService.RegisterLineEngine for flatfee and usagebased — if BillingService does not support this, construction fails. (`svcs, err := testutils.NewServices(t, testutils.Config{
    Client: dbClient, BillingService: billingSvc,
    FlatFeeHandler: handlers.FlatFee, ...
})`)
**Config.Validate() guards all required dependencies** — Config.Validate() returns joined errors for any nil required field (Client, BillingService, FeatureService, StreamingConnector, and all three handlers). RecognizerService defaults to recognizer.NoopService{} if nil. (`if config.RecognizerService == nil { config.RecognizerService = recognizer.NoopService{} }
if err := config.Validate(); err != nil { return nil, err }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handlers.go` | Mock implementations of flatfee.Handler, creditpurchase.Handler, usagebased.Handler with compile-time interface assertions and newMockLedgerTransactionGroupReference() returning fresh ULIDs. | When a new method is added to any Handler interface, a corresponding no-op must be added here or the mock stops compiling. Each callback returns a fresh ULID — tests that verify specific IDs must capture the return value. |
| `service.go` | Config struct, Validate(), Services struct, and NewServices factory that constructs the full charges sub-service tree for tests. | NewServices registers flatfee and usagebased line engines on BillingService via RegisterLineEngine. CreditPurchase line engine is also registered. Adding a new charge type requires adding its adapter, service, and line engine registration here. |

## Anti-Patterns

- Importing app/common from testutils — creates import cycles; build all dependencies from package constructors directly
- Adding stateful side effects to mock handlers without a Reset() method — causes test pollution across suite test cases
- Using NewServices in production wiring (app/common) — this factory is test-only and uses mock handlers with no real ledger integration
- Sharing a single MockHandlers instance across parallel tests — the mock handlers are stateless so sharing is safe, but any future state added must account for concurrency

## Decisions

- **MockHandlers returns a new ULID on every ledger callback instead of a fixed constant** — Tests that verify ledger transaction group IDs need unique, distinguishable IDs per callback invocation; a fixed constant would make it impossible to differentiate which callback ran or how many times.
- **NewServices wires all sub-services rather than requiring callers to compose them** — The charges stack has a deep dependency tree (meta adapter, locker, lineage, three charge-type stacks); centralizing setup in one factory reduces test boilerplate and ensures consistent wiring across all integration tests.

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
