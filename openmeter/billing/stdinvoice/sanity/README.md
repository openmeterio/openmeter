# Invoice sanity checks

Sanity validation checks whether calculated invoice totals agree with the detail
consumed by invoicing apps. It is read-only and does not rate charges, repair
missing data, or establish that the commercial intent was correct. `Validate`
checks any supplied invoice regardless of its lifecycle status; callers own
when to invoke it.

Callers must supply expanded invoice lines. A loaded empty collection is valid;
unloaded lines and nil entries are critical validation issues. Checks reconcile:

- detailed discount and credit totals with their breakdowns, using billing's
  per-entry currency rounding (including signed discount rounding corrections);
- category placement: regular details use `Amount`, commitments use
  `ChargesTotal`, and the other gross field must be zero;
- payable totals with amount plus charges plus exclusive tax minus discounts
  and credits, at invoice, parent, and detail levels;
- every parent total component with its active detailed-line sum, and every
  invoice total component with its active parent-line sum;
- parent credit allocations with active details grouped by realization ID, allowing
  one realization to span several details;
- positive credit entries with realization IDs and allocation limits against the
  detail's pretax balance (`Amount + ChargesTotal - DiscountsTotal`);
- matching parent/invoice currency and currency precision for all eight calculated
  total fields at invoice, parent and detail levels;
- persisted invoice, parent, detail and amount-discount IDs, unique within each
  entity kind; parent/detail invoice IDs and namespaces must match the invoice;
  detail child references must be present and unique within their parent;
- ordered parent/detail service periods and the invoice's active-parent envelope;
- nonnegative parent payable totals and required `UsageBased`/`Price` pointers,
  including flat-price parents.

Inclusive tax is already in gross value and is not added again. Comparisons are
exact: validation does not round stored totals to hide discrepancies or impose
nonnegative quantities/totals on signed correction details. Deleted parents,
details and discount entries do not contribute. Zero-value parents without
stored details remain valid, including provisional credit-purchase lines.
Missing or nil parents prevent invoice aggregation from being evaluated, while
independent checks still report their own issues. Invalid currency is reported
once and skips only checks that require currency rounding.

An invoice with no active parents must have no service period. Point intervals
are valid under the existing period contract. Detail periods need not be contained
in their parent: corrections can span different intervals. Provider IDs remain
optional before synchronization. Credit limits use actual allocations, not a
possibly stale `CreditsTotal`, and exclusive tax cannot increase that budget.
Calculated totals must already be currency-rounded; unit prices, quantities and
allocation input amounts retain their existing precision and rounding order.
Gross amounts are not checked against unit price times quantity: progressive
billing subtracts already-rounded cumulative charges, so a valid delta can differ
from the rounded product of its own quantity and unit price.

This does not prove ledger credit ownership, commercial pricing intent, or
provider request correctness. Supplier and standard-line
`InvoiceAt` are outside sanity validation. Existing lifecycle and enforcement
behavior applies to every check.

Failures use the `billing.invoice.sanity` component and preserve individual issue
codes, field paths and reconciliation values. The checker is separate from
ordinary model validation so incomplete historical records remain readable.
The invoice state machine invokes checks at app validation, synchronization,
finalization and asynchronous readiness call sites, all during draft and
issuing. Issuing checks again after line finalization and before each external
call, including retries. `IssuedAt` does not disable checks: it is set before
app finalization.

Post-advance hooks do not repeat sanity validation, including when reached
before issuance. Calculated invoice contents are immutable from `issued`
onward, and these checks must not block later payment processing. Deletion
also bypasses sanity validation so inconsistent invoices can still be cleaned
up.

`billing.enforceInvoiceSanityChecks` remains a service boolean, defaulting to
`false`. The state machine maps it to `enforcement_reporting_only` or
`enforcement_enabled`. Reporting-only mode logs issues at error severity and
allows the app call. Enabled mode also returns validation errors: mutating
callers persist them through existing lifecycle failure handling, while real
readiness guards withhold the app call without persisting issues or changing
state. System errors propagate in both modes.

Status-details prediction temporarily uses `enforcement_disabled`, which skips
`Validate` entirely, and restores the previous mode on return. Prediction may
pass partial invoices to app readiness methods; it describes lifecycle
possibilities, not whether real execution will pass sanity validation. Partial
reads therefore retain their requested expansion without false sanity logs or
truncated action predictions. Actual execution still validates before app
handoffs.

An allowed mutating app call clears stale sanity issues while preserving
unrelated issues. Sanity checks run at app call sites; advancement does not
perform a separate check.
