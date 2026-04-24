# testutils

<!-- archie:ai-start -->

> Shared test infrastructure for the charges domain: provides MockHandlers (no-op implementations of flatfee.Handler, creditpurchase.Handler, usagebased.Handler) and a NewServices factory that wires the full charges stack from raw dependencies for use in integration tests.

## Patterns

**MockHandlers as zero-side-effect handler impls** — mockFlatFeeHandler, mockCreditPurchaseHandler, and mockUsageBasedHandler implement all handler interface methods; ledger callbacks return a fresh ulid-based ledgertransaction.GroupReference so tests can verify IDs without a real ledger. (`handlers := testutils.NewMockHandlers()
flatFeeService, _ := flatfeeservice.New(flatfeeservice.Config{Handler: handlers.FlatFee, ...})`)
**NewServices wires the full charges stack** — NewServices(t, Config) constructs meta adapter, locker, lineage adapter/service, each charge-type adapter/service, line engines, and the top-level charges.Service in one call — mirrors production wiring in app/common without importing it. (`svcs, err := testutils.NewServices(t, testutils.Config{
	Client:              dbClient,
	BillingService:      billingSvc,
	FeatureService:      featureSvc,
	StreamingConnector:  mockConnector,
	FlatFeeHandler:      handlers.FlatFee,
	CreditPurchaseHandler: handlers.CreditPurchase,
	UsageBasedHandler:   handlers.UsageBased,
})`)
**Config.Validate() guards all required dependencies** — Config.Validate() returns joined errors for any nil required field (Client, BillingService, FeatureService, StreamingConnector, and all three handlers); callers must handle the error before using Services. (`if err := config.Validate(); err != nil { return nil, err }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handlers.go` | Mock implementations of flatfee.Handler, creditpurchase.Handler, usagebased.Handler. Each callback returns a deterministic ledgertransaction.GroupReference via newMockLedgerTransactionGroupReference(). | When a new handler method is added to any Handler interface, a corresponding no-op must be added here or the mock will stop compiling. |
| `service.go` | Config struct, Validate(), Services struct, and NewServices factory function that builds the full charges sub-service tree for tests. | NewServices registers flat-fee and usage-based line engines on BillingService via RegisterLineEngine — if BillingService does not support RegisterLineEngine, construction will fail. |

## Anti-Patterns

- Importing app/common from testutils — creates import cycles; build all dependencies from package constructors directly
- Adding stateful side effects to mock handlers without a Reset() method — test pollution across suite test cases
- Using NewServices in production wiring (app/common) — this factory is test-only and uses mock handlers

## Decisions

- **MockHandlers returns a new ulid on every ledger callback instead of a fixed constant** — Tests that verify ledger transaction group IDs need unique, distinguishable IDs per callback invocation; a fixed constant would make it impossible to differentiate which callback ran.

<!-- archie:ai-end -->
