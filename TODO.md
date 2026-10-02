# TODO

The issues below are grouped by the lifecycle boundary and regression cases worth fixing together. Each distinct failure keeps its own test; charge types are variants within that test. The regressions demonstrate service-level failures, not additional affected invoices.

## 1. Issuance callbacks: deleted lines and persisted partial progress

Fix deleted-line dispatch during `invoice_issued` and persisted partial preparation during `invoice_finalizing`. The completed `invoice_issued` retry is handled by the base branch and is covered there for flat-fee and usage-based charges.

### Deleted invoice lines are dispatched during issuance

- [ ] Exclude deleted invoice lines from `invoice_issued` dispatch.

An invoice retains a deleted line whose charge cannot accept `invoice_issued`. Dispatching that line fails with `unsupported operation` and leaves the invoice in `issuing.charge_booking_failed`, blocking its live lines.

Reproducer: `TestIssuingSkipsDeletedChargeLine` in [issuing tests](test/credits/credit_then_invoice_issuing_test.go), with flat-fee and usage-based variants.

### Finalization retry replays completed line preparation

- [ ] Resume finalization from persisted preparation without recreating or replacing an already prepared run.

The first charge reaches `active.realization.issuing` before the second line's preparation fails. The invoice enters `issuing.line_finalization_failed`. Retry sends `invoice_finalizing` to the already prepared charge, which returns `unsupported operation`; issuance cannot finish. This failure occurs before issuance accrual booking and is distinct from replaying `invoice_issued`.

Reproducer: `TestInvoiceFinalizationRetryPreservesCompletedLinePreparation` in [retry tests](test/credits/credit_then_invoice_retry_test.go), with flat-fee and usage-based variants.

### Recovery constraint: later correction drift

- [ ] Account for unsupported correction history when recovering a previously booked invoice.

A later shrink can mark a booked run `invalid_due_to_unsupported_credit_note` while the immutable invoice retains the line and ledger booking. Recovery must preserve that history. This drift cannot explain the initial booking failure, and the retry tests do not establish a correction policy.

Passing characterization: `TestIssuingFailedInvoicePreservesUnsupportedCorrectionHistory` in [issuing tests](test/credits/credit_then_invoice_issuing_test.go). It confirms preservation of invoice history and accounting, not a separate demonstrated booking failure.

## 2. Payment booking: resume settlement from persisted progress

The base branch makes authorization retry-safe across flat-fee, usage-based, and invoice-funded credit-purchase charges. Settlement helpers and charge transitions still reject existing settlements, and a completed settlement reported alongside a callback failure still leaves the invoice unable to recover.

- [ ] Recognize a matching persisted settlement before replaying the corresponding booking or charge transition.
- [ ] Preserve payment identity, realization identity, and original ledger references; book only missing work.
- [ ] Validate that completed work belongs to the same invoice, line, and amount rather than treating every rejected trigger as success.
- [ ] Allow an invoice whose payments already settled to reach `paid`.

The journal does not deduplicate repeated bookings. Retry safety must use persisted lifecycle facts before issuing accounting effects; an invoice-status-only fix would leave the underlying retry contract broken.

| Distinct issue | Persisted state before retry and current failure | Expected recovery | Reproducer and variants |
| --- | --- | --- | --- |
| Partial settlement | Both lines authorized, first already settled. `payment_processing.booking_settled_failed` retries the first settlement and gets `payment already settled` for flat-fee/usage or `unsupported operation` for credit purchases. | Settle only the remaining line; reach `paid`. | `TestPaymentSettlementRetryPreservesCompletedLineBooking` in [retry tests](test/credits/credit_then_invoice_retry_test.go): all three charge types. |
| Combined booking already completed settlement | Real settlement completes before a one-time callback failure is reported. The invoice is in `payment_processing.booking_authorized_and_settled_failed` with settled payments. Retry safely recognizes authorization but replays settlement, getting `payment already settled` for flat-fee/usage or `unsupported operation` for credit purchases. | Reach `paid` with the same payments and no new ledger transactions. | `TestPaymentBookingRetryPreservesCompletedSettlement` in [payment tests](test/credits/credit_then_invoice_payment_test.go): all three charge types. |

These cases use real service calls and one-time callback faults, without SQL updates or adapter edits to manufacture state. They reload the charge facts and follow payment references to the journal, checking posting amounts, accounting stages, provenance, and transaction preservation. The original $21 flat-fee case also checks the complete accrual/authorization/settlement references and balances. Existing bookings remain intact on retry, but the invoice cannot recover and missing work remains unbooked.

The two remaining payment cases plus the finalization case cover eight charge-type variants, all intentionally failing on recovery. `TestCombinedPaymentBookingRetryResumesAfterAuthorization` now passes on the base branch, confirming that completed authorization can resume into settlement. The existing direct-paid and partial-credit payment lifecycle tests, and the correction-history characterization, also pass.

## 3. Provider errors: reconcile superseded failures before evaluating success

This is a separate validation-issue lifecycle problem at the billing app/state-machine boundary. It can strand an invoice even when no callback fault occurs and every payment booking succeeds; payment retry fixes alone do not remove the stale error.

- [ ] Reconcile superseded collection errors before deciding whether a successful paid callback failed. Check both `initiatePayment` and `trigger_invoice` component histories while preserving unrelated unresolved errors.
- [ ] Prevent an old provider failure from being copied into the successful paid callback's component.

A $21 flat-fee charge is issued, the provider reports a critical payment failure through `BillingService.TriggerInvoice`, then reports paid through the same service. Authorization and settlement complete, but `FireAndActivate` checks the old critical issue after booking and leaves the invoice in `payment_processing.booking_authorized_and_settled_failed`. The old issue also appears in the paid callback's component.

Reproducer: `TestPaidCallbackClearsPreviousPaymentFailureBeforeBooking` in [payment tests](test/credits/credit_then_invoice_payment_test.go). Its accounting assertions pass; invoice recovery intentionally fails.

The test uses a custom-invoicing app and the generic payment-app boundary used by Stripe. The failure is attributed to `initiatePayment`, matching Stripe's collection-error mapping. This proves a reachable service-level sequence. The exact historical webhook sequence remains unverified; current main rejects unavailable duplicate failure triggers.

## 4. Diagnostics and monitoring: expose the failed boundary and durable progress

Fix error context and monitoring together so an internal failure identifies the callback, charge state, and completed accounting work needed for triage.

### Persisted errors lose root-cause context

- [ ] Preserve `invoice_issued` in the validation message and include `trigger`, `charge_id`, and `charge_status` attributes.
- [ ] Add invoice and invoice-line ID attributes at the line-engine boundary.

Ordinary error wrappers lose their added context during validation conversion, leaving only `unsupported operation` and no attributes. This is a diagnostic gap, not another booking failure. Use existing validation-context wrappers or structured issues when fixing the owning boundary; inspect converter consumers before changing its contract.

Failing reproducer: `TestMachine_UnsupportedInvoiceIssuedErrorPreservesValidationContext` in [machine_test.go](openmeter/billing/charges/statemachine/machine_test.go). The test also records the missing invoice-line context follow-up.

### Monitor internal invoice error states

- [ ] Monitor `issuing.line_finalization_failed`, `issuing.charge_booking_failed`, `payment_processing.booking_authorized_failed`, `payment_processing.booking_settled_failed`, and `payment_processing.booking_authorized_and_settled_failed`.
- [ ] Track affected-invoice counts, time spent in failed states, repeated retries, and validation issue codes/components. Surface new failures and invoices that remain stuck.
- [ ] Include charge lifecycle state and completed realization/payment ledger references in investigation tooling so partial progress is visible.
