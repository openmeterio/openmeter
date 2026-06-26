# testutils

<!-- archie:ai-start -->

> Reusable test wiring for the full charges stack. NewServices assembles meta/lineage/flatfee/usagebased/creditpurchase adapters+services and the root charges service from external deps; MockHandlers supplies no-op ledger-transaction handlers. Consumed by subscriptionsync and cross-package billing/credit tests.

## Patterns

**One-call stack assembly** — NewServices(t, Config) constructs every adapter and service in dependency order (metaAdapter, lockr, lineage, flatfee, usagebased, creditpurchase, root charges) and registers each line engine on BillingService. (`if err := config.BillingService.RegisterLineEngine(flatFeeService.GetLineEngine()); err != nil { return nil, ... }`)
**Mock handlers return synthetic ledger refs** — MockHandlers implements flatfee/creditpurchase/usagebased Handler interfaces, each On* method returning newMockLedgerTransactionGroupReference() (a fresh ULID) so charge lifecycle runs without real ledger writes. (`func newMockLedgerTransactionGroupReference() ledgertransaction.GroupReference { return ledgertransaction.GroupReference{TransactionGroupID: ulid.Make().String()} }`)
**Config.Validate gates required deps** — Config.Validate collects errors for missing Client/BillingService/FeatureService/StreamingConnector/handlers/TaxCodeService via errors.Join; RecognizerService defaults to recognizer.NoopService and Logger defaults to slog.Default(). (`if config.RecognizerService == nil { config.RecognizerService = recognizer.NoopService{} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Config + Validate, Services struct, NewServices full-stack builder over an *entdb.Client. | Line engines MUST be registered before charge operations; RatingService uses billingratingservice.New() per engine; logger/recognizer are the only optional fields. |
| `handlers.go` | MockHandlers and the three mock Handler implementations returning synthetic ledger group references. | mockFlatFeeHandler.OnAllocateCredits returns nil for zero PreTaxAmountToAllocate; these mocks do no real allocation accounting beyond echoing amounts. |

## Anti-Patterns

- Hand-wiring charge services in tests instead of calling NewServices.
- Skipping RegisterLineEngine, which leaves billing unable to project charge lines.
- Using the mock handlers when a test needs real ledger/recognizer accounting.

## Decisions

- **A single NewServices builder mirrors production wiring (base_test.go) for cross-package consumers.** — subscriptionsync and stripe/credits tests need the whole charges stack without duplicating the multi-adapter construction sequence.

<!-- archie:ai-end -->
