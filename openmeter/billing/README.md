# Billing

Billing turns billable work into fiat invoices and advances those invoices
through calculation, external invoicing, and payment.

It owns:

- billing profiles and customer-specific workflow overrides
- gathering and standard invoice aggregates
- invoice-line identity and persistence
- invoice calculation, validation, and lifecycle orchestration
- coordination with tax, invoicing, and payment apps

It does not own the subscription schedule that produces billable work, or the
charge and ledger semantics behind a charge-backed line. Subscription
reconciliation is described in
[Subscription sync](worker/subscriptionsync/README.md); charge behavior is
described in [Charges](charges/README.md).

## Billing configuration becomes invoice history

A billing profile selects the workflow and the apps responsible for tax,
invoicing, and payment. A namespace has one default profile. A customer
override may select another profile and replace supported parts of its
workflow.

The merged customer configuration is resolved when a standard invoice is
created. The invoice keeps the customer, supplier, workflow configuration, and
app references needed to finish its lifecycle. Customer-visible invoice
history must not retrospectively follow later customer or profile changes.

## Gathering and standard invoices

A **gathering invoice** is mutable staging for future invoice lines. There is at
most one per customer and fiat currency. It does not advance through the
standard invoice state machine and is deleted when it has no remaining lines.

A **standard invoice** is created by collecting eligible gathering lines for
one customer and currency. The collected line identities are preserved while
their gathering representations are replaced by standard lines. The standard
invoice then owns the calculated and externally synchronized record presented
to the customer.

An invoice's lines may be absent because they were not expanded. This is
different from a loaded but empty line collection.

A standard line is the billable item on the invoice. Detailed lines explain
the calculated price components beneath it, such as tier or flat-fee
components. They are derived calculation output, not another source of
subscription intent.

## Time has distinct meanings

- a line's service period is when the service was provided
- `InvoiceAt` is when the line becomes eligible to be invoiced
- a gathering invoice's next collection time is when the collector should
  reconsider its pending lines
- a standard invoice's collection time is the cutoff used while snapshotting
  quantities and completing collection

Collection alignment may move the effective collection cutoff to an anchor.
Do not use one of these timestamps as a substitute for another.

Progressive billing may split a service period across invoices. The split-line
group preserves the full service-period identity while its children represent
the portions materialized on individual invoices. Already immutable invoice
history is not rewritten to make a later subscription change look as if it had
always existed.

## Line ownership

Billing owns the invoice aggregate, line IDs, and persistence. A line's engine
owns the line-specific calculations and lifecycle side effects for the lines
routed to it. The built-in invoicing engine handles billing-native lines;
charge engines handle charge-backed lines.

Engine callbacks operate on groups of lines. When a callback replaces lines,
billing requires it to preserve the input line IDs before accepting the
result. API-originated line edits and system-originated reconciliation are
different change sources: API edits may change manual ownership, while system
edits preserve the ownership contract of their source.

## Invoice lifecycle and failure

The standard invoice state machine coordinates:

1. collection and quantity snapshotting
2. calculation and validation
3. draft synchronization with the invoicing app
4. approval and issuing
5. charge booking and payment processing

When issuing starts, a standard invoice with no non-deleted lines is deleted
instead of finalized. Billing records this as a system deletion, synchronizes
the deletion with the invoicing app, and skips charge booking and payment. This
decision is based on line presence rather than monetary totals: a non-deleted
line with a zero total still follows the normal issuing lifecycle.

Critical validation issues stop advancement. External app and line-engine
failures leave the invoice in an explicit failed state so the failed step can
be retried without replaying stable lifecycle states. Issuing and payment
booking therefore happen in retryable intermediary states; authorization is
booked before settlement when a payment provider reports both at once.

If quantity snapshotting discovers that a persisted feature no longer has its
required meter association, collection still materializes the standard invoice
in `draft.invalid_created` with the critical validation code
`invoice_line_feature_has_no_meters`. The affected quantities stay
unsnapshotted. Retry from this state re-enters `draft.created` for calculation
and validation before returning through collection, so the repaired meter is
queried again; repeated collection failures return to the same state.
Post-collection validation failures use `draft.invalid` and retry validation
instead. Operational snapshot failures abort collection instead of persisting
an incomplete invoice.

Before external invoice finalization, billing invokes each line engine's line
finalization callback. Engines return fully calculated lines with unchanged
line IDs; billing replaces those lines, sums their totals, and persists the
invoice before sending it to the invoicing app. Line finalization has its own
retryable failure state, so an external app retry does not repeat stable
engine-owned effects. Flat-fee and usage-based accounting preparation complete
at this boundary; the later external-issued callback validates and advances the
already-prepared charge lifecycle. Line-finalization failures and subsequent
issuing are retry-only. If invoicing app synchronization fails after durable
preparation succeeds, charge-owned cleanup must correct that preparation before
the invoice can be deleted.

A deleted app keeps its historical identity and can still be expanded on
invoices. Its generic app operations return `app.ErrAppDeleted`, while its
invoicing operations return billing-owned validation issues. Critical issues
move the invoice into the failed state for that step. Before issuing, invoice
deletion returns a warning when provider cleanup has to be skipped.

Retrying and deleting are separate domain actions. Use the billing service's
retry operation for a retryable failure and its delete operation for the
delete lifecycle; firing a generic state-machine trigger bypasses preparation
and audit semantics owned by those operations.

Billing publishes created and updated standard-invoice snapshots for
downstream consumers. [Notifications](../notification/README.md) maps those
snapshots into its own API-shaped historical payloads and delivers them
asynchronously; webhook delivery is not part of the invoice lifecycle. Every
snapshot contains the base identity of all three workflow apps. Building that
identity does not instantiate provider implementations; provider-specific
event data is included only when the implementation is already available to
the lifecycle operation.

## Consistency invariants

- invoice manipulation is serialized per customer and runs under the billing
  transaction and customer update lock
- every invoice contains one fiat currency
- a customer cannot have two active gathering invoices for the same currency
- pending-line creation resolves every supplied feature; metered prices require the feature's meter
- gathering-to-standard conversion preserves line IDs
- line engines cannot silently change the identity or count of callback output
- immutable invoice drift is recorded as validation issues rather than
  rewriting customer-visible history
- an app referenced by a non-final invoice cannot be uninstalled, because its
  provider-specific resources are required to complete that invoice's
  snapshotted workflow
