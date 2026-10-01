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
