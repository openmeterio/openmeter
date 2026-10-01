# TODO

## Monitor internal error states

- [ ] Monitor invoice lifecycle failures, including `issuing.charge_booking_failed` and `payment_processing.booking_authorized_and_settled_failed`.
- [ ] Track affected-invoice counts, time spent in the failed state, repeated retries, and validation issue codes/components. Surface new failures and invoices that remain stuck.
- [ ] Include charge lifecycle state and completed accounting facts in investigation tooling so partial progress is visible.

## Findings

### 1. Invoice-issued callbacks mishandle deleted lines and partial progress

- [ ] Skip deleted invoice lines when dispatching `invoice_issued`.
- [ ] Make retry recognize matching completed issuance work and continue remaining bookings without replaying accounting effects.

Both mechanisms are reproduced for flat-fee and usage-based charges. The retry reproducer uses a one-time callback fault, so it is independent of the deleted-line defect.

Tests in [credit_then_invoice_issuing_test.go](test/credits/credit_then_invoice_issuing_test.go):

- `TestIssuingSkipsDeletedChargeLine`
- `TestIssuingRetryPreservesCompletedChargeBooking`

Later correction drift is covered separately by `TestIssuingFailedInvoicePreservesUnsupportedCorrectionHistory`. Shrinking an already-booked charge can mark its run `invalid_due_to_unsupported_credit_note` while preserving the invoice line and ledger booking. This passing characterization does not establish a recovery policy.

### 2. Stored validation issues lose issuance context

- [ ] Preserve `invoice_issued` in the validation message and include `trigger`, `charge_id`, and `charge_status` attributes.
- [ ] Add invoice and invoice-line ID attributes at the line-engine boundary.

Ordinary error wrappers currently lose their context during validation conversion, leaving only `unsupported operation` and no attributes. Use the existing validation context wrappers or structured issues when implementing the fix; investigate the owning boundary before changing the converter contract.

Reproducer: `TestMachine_UnsupportedInvoiceIssuedErrorPreservesValidationContext` in [machine_test.go](openmeter/billing/charges/statemachine/machine_test.go). It currently fails on the missing event message and attributes; the test also contains the invoice-line context TODO.

### 3. Previous provider failures survive successful payment booking

- [ ] Reconcile previous collection errors before deciding whether a successful paid callback failed. Check both `initiatePayment` and `trigger_invoice` component histories.

`TestPaidCallbackClearsPreviousPaymentFailureBeforeBooking` in [credit_then_invoice_payment_test.go](test/credits/credit_then_invoice_payment_test.go) creates and issues a $21 flat-fee charge, reports a critical payment failure through `BillingService.TriggerInvoice`, then reports paid through the same service. Authorization and settlement complete, but `FireAndActivate` checks the old critical issue after booking and leaves the invoice in `payment_processing.booking_authorized_and_settled_failed`. The old issue is also copied into the paid callback's component.

The test uses a custom-invoicing app and the generic payment-app boundary used by Stripe. Its failure event attributes the issue to `initiatePayment`, matching Stripe's collection-error mapping. It proves a reachable service-level sequence, not the exact historical webhook sequence: current main rejects unavailable duplicate failure triggers, while the older recorded issue came from that trigger path.

### 4. Payment booking retries reject completed settlement

- [ ] Recognize a matching persisted authorization or settlement on retry, retain its original ledger references, and resume remaining work without adding transactions.

`TestPaymentBookingRetryPreservesCompletedSettlement` in [credit_then_invoice_payment_test.go](test/credits/credit_then_invoice_payment_test.go) delegates to the real settlement callback before reporting a one-time callback failure. The failed invoice and settled payment survive reloading. `RetryInvoice` then replays authorization and returns `payment_already_authorized`, even though the payment is settled. This reproducer is independent of stale provider issues.

Both tests use service calls rather than SQL updates or adapter edits to manufacture state. Their accounting assertions pass: the immutable run has the correct invoice, line, service period, amount, and three distinct ledger group references; the referenced USD postings are balanced and match invoice accrual, authorization, and settlement; open and authorized receivables are cleared. Retry preserves the payment and adds no ledger transactions, but fails to advance the invoice. Both regression tests intentionally fail on invoice recovery; the existing flat-fee payment lifecycle test passes.
