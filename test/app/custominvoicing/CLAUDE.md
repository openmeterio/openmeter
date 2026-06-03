# custominvoicing

<!-- archie:ai-start -->

> Integration tests for the custom invoicing app validating the async sync protocol (draft→issuing→payment) and invoice event JSON marshaling against a real Postgres DB via billingtest.BaseSuite. All service interactions must go through BillingService and CustomInvoicingService — never adapter or Ent directly.

## Patterns

**Embed billingtest.BaseSuite** — Test suites embed billingtest.BaseSuite (not app/common wiring) to get BillingService, CustomerService, AppService, MockStreamingConnector from raw constructors. (`type CustomInvoicingTestSuite struct { billingtest.BaseSuite }`)
**setupDefaultBillingProfile helper** — Each scenario calls s.setupDefaultBillingProfile(ctx, namespace, config) to install/configure the app and provision the billing profile. DraftPeriod is P0D so lines become invoiceable immediately. (`s.setupDefaultBillingProfile(ctx, namespace, appcustominvoicing.Configuration{EnableDraftSyncHook: true, EnableIssuingSyncHook: true})`)
**Drive state machine via service layer only** — Invoice lifecycle is driven exclusively through s.BillingService and s.CustomInvoicingService (SyncDraftInvoice, SyncIssuingInvoice, HandlePaymentTrigger) — never adapter or Ent. (`s.CustomInvoicingService.SyncDraftInvoice(ctx, appcustominvoicing.SyncDraftInvoiceInput{InvoiceID: invoice.GetInvoiceID(), UpsertInvoiceResults: upsertResults})`)
**Unique namespace per test function** — Every test function uses a unique string literal namespace to avoid cross-test DB contamination without teardown. (`namespace := "ns-custom-invoicing-flow"`)
**MockStreamingConnector with deferred Reset** — Inject usage via AddSimpleEvent before invoicing; always defer s.MockStreamingConnector.Reset() to prevent stale events bleeding into other tests. (`s.MockStreamingConnector.AddSimpleEvent("test", 100, periodStart.Add(time.Minute)); defer s.MockStreamingConnector.Reset()`)
**Suite.Run for sub-scenarios** — Group related lifecycle steps with s.Run('description', func(){}) to share invoice state and produce hierarchical output. (`s.Run("invoice can be created", func() { invoices, err := s.BillingService.InvoicePendingLines(...) })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `invocing_test.go` | Main integration test covering full invoice flow (hooks enabled) and payment-status-only flow (hooks disabled); provides setupDefaultBillingProfile used by event_test.go. | DraftPeriod is P0D so lines are immediately invoiceable — changing it requires adjusting AsOf in InvoicePendingLines. Hooks-enabled yields draft.syncing; hooks-disabled yields payment_processing.pending directly. |
| `event_test.go` | Unit JSON round-trip test for billing.StandardInvoiceCreatedEvent — validates app bases and appcustominvoicing.Meta survive marshal/unmarshal. | Uses context.Background() rather than t.Context() — acceptable since no cancellation is tested; new tests should prefer t.Context(). |

## Anti-Patterns

- Calling adapter or Ent methods directly instead of going through BillingService / CustomInvoicingService.
- Importing app/common wiring into test setup — use raw constructors via billingtest.BaseSuite.
- Sharing namespace strings across test functions — always use a unique namespace per top-level test.
- Forgetting defer s.MockStreamingConnector.Reset() — stale events bleed into subsequent tests.
- Setting DraftPeriod to a non-zero duration without adjusting AsOf — lines won't be invoiceable.

## Decisions

- **Tests embed billingtest.BaseSuite instead of constructing services manually.** — Keeps setup DRY and independent of app/common to avoid import cycles; BaseSuite provides shared services from underlying constructors.
- **Both EnableDraftSyncHook=true and =false flows are exercised in separate test functions.** — Custom invoicing has two distinct state machine paths (full async sync vs payment-status-only); both need explicit coverage.

## Example: Full invoice lifecycle with sync hooks enabled

```
import (
  appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
  "github.com/openmeterio/openmeter/openmeter/billing"
  billingtest "github.com/openmeterio/openmeter/test/billing"
)

type MyTestSuite struct { billingtest.BaseSuite }

func (s *MyTestSuite) TestFlow() {
  ctx := context.Background()
  namespace := "ns-my-unique-test"
  s.setupDefaultBillingProfile(ctx, namespace, appcustominvoicing.Configuration{EnableDraftSyncHook: true})
  invoices, _ := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{...})
  s.Equal(billing.StandardInvoiceStatusDraftSyncing, invoices[0].Status)
  synced, _ := s.CustomInvoicingService.SyncDraftInvoice(ctx, appcustominvoicing.SyncDraftInvoiceInput{InvoiceID: invoices[0].GetInvoiceID()})
// ...
```

<!-- archie:ai-end -->
