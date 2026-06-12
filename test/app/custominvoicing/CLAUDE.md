# custominvoicing

<!-- archie:ai-start -->

> Integration tests for the custom-invoicing app (openmeter/app/custominvoicing) verifying that draft/issuing sync hooks and payment-status triggers drive a billing invoice through its state machine. Tests are built on billingtest.BaseSuite, not the application wiring layer.

## Patterns

**Embed billingtest.BaseSuite** — Test suites embed billingtest.BaseSuite, exposing services like AppService, CustomerService, MeterAdapter, FeatureService, BillingService, CustomInvoicingService, MockStreamingConnector. CustomInvoicingEventTestSuite further embeds CustomInvoicingTestSuite to reuse setup. (`type CustomInvoicingTestSuite struct { billingtest.BaseSuite }`)
**Shared profile setup helper** — setupDefaultBillingProfile installs the custom-invoicing app via AppService.InstallMarketplaceListing (app.AppTypeCustomInvoicing), pushes appcustominvoicing.Configuration through AppService.UpdateApp, then ProvisionBillingProfile. Toggle EnableDraftSyncHook / EnableIssuingSyncHook per scenario. (`s.setupDefaultBillingProfile(ctx, namespace, appcustominvoicing.Configuration{EnableDraftSyncHook: true, EnableIssuingSyncHook: true})`)
**Drive invoice lifecycle through service calls** — Create gathering lines via BillingService.CreatePendingInvoiceLines, materialize via InvoicePendingLines, then advance with CustomInvoicingService.SyncDraftInvoice, SyncIssuingInvoice, HandlePaymentTrigger; assert exact billing.StandardInvoiceStatus* after each. (`draftSyncedInvoice, err := s.CustomInvoicingService.SyncDraftInvoice(ctx, appcustominvoicing.SyncDraftInvoiceInput{InvoiceID: invoice.GetInvoiceID(), UpsertInvoiceResults: upsertResults})`)
**Builder results for sync inputs** — Sync inputs are constructed with the billing fluent builders billing.NewUpsertStandardInvoiceResult() and billing.NewFinalizeStandardInvoiceResult() with chained SetInvoiceNumber/SetExternalID/SetPaymentExternalID/AddLineExternalID. (`billing.NewUpsertStandardInvoiceResult().SetInvoiceNumber("DRAFT-123").SetExternalID("ext-123").AddLineExternalID(invoice.Lines.OrEmpty()[0].ID, "ext-123")`)
**Per-test unique namespace string** — Each test hardcodes a descriptive namespace string (e.g. "ns-custom-invoicing-flow") rather than ULIDs, and uses context.Background() at the top of the test body. (`namespace := "ns-custom-invoicing-flow"`)
**Meter/streaming teardown via defer** — Tests that set up meters and stream events clean up with deferred s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{}) and s.MockStreamingConnector.Reset(). (`defer s.MockStreamingConnector.Reset()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `invocing_test.go` | Main suite (note misspelled filename). Defines CustomInvoicingTestSuite, setupDefaultBillingProfile, and full lifecycle tests TestInvoicingFlowHooksEnabled and TestInvoicingFlowPaymentStatusOnly. | Filename is 'invocing' not 'invoicing'. With hooks disabled, InvoicePendingLines lands directly in PaymentProcessingPending and assigns a generic number like INV-TECU-1; with hooks enabled it lands in DraftSyncing and requires explicit SyncDraftInvoice. |
| `event_test.go` | CustomInvoicingEventTestSuite verifying billing.NewStandardInvoiceCreatedEvent is JSON round-trippable and carries app bases (Tax/Payment/Invoicing) plus Meta decoded via appcustominvoicing.Meta.FromEventAppData. | Asserts event.Apps.*.AppBase equals invoice.Workflow.Apps.*.GetAppBase(); Meta.Configuration flags must survive marshal/unmarshal. |

## Anti-Patterns

- Calling app/billing adapters directly or importing app/common wiring instead of building from BaseSuite-provided services.
- Asserting invoice progress without checking the exact billing.StandardInvoiceStatus* enum at each step.
- Leaving meters or streamed events in place across tests (always defer ReplaceMeters({}) and MockStreamingConnector.Reset()).
- Assuming hooks-disabled and hooks-enabled flows produce the same terminal status — they diverge at InvoicePendingLines.

## Decisions

- **Tests exercise CustomInvoicingService.SyncDraftInvoice/SyncIssuingInvoice/HandlePaymentTrigger rather than simulating webhook callbacks.** — Validates the real state-machine transitions and external-ID propagation the custom-invoicing app guarantees to integrators.
- **Payment triggers are validated for full-mesh legality (e.g. paid cannot transition to uncollectible), expecting billing.ValidationError.** — Ensures invalid status transitions surface as ValidationError rather than silent state corruption.

## Example: Advance a custom-invoicing invoice from draft.syncing to issuing.syncing

```
upsertResults := billing.NewUpsertStandardInvoiceResult().
	SetInvoiceNumber("DRAFT-123").
	SetExternalID("ext-123").
	AddLineExternalID(invoice.Lines.OrEmpty()[0].ID, "ext-123")

draftSyncedInvoice, err := s.CustomInvoicingService.SyncDraftInvoice(ctx, appcustominvoicing.SyncDraftInvoiceInput{
	InvoiceID:            invoice.GetInvoiceID(),
	UpsertInvoiceResults: upsertResults,
})
s.NoError(err)
s.Equal(billing.StandardInvoiceStatusIssuingSyncing, draftSyncedInvoice.Status)
```

<!-- archie:ai-end -->
