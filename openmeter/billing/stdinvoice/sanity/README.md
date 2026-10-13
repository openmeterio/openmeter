# Invoice sanity checks

Sanity validation checks whether calculated invoice totals agree with the detail
consumed by invoicing apps. It is read-only and does not rate charges, repair
missing data, or establish that the commercial intent was correct. `Validate`
checks any supplied invoice regardless of its lifecycle status; callers own
when to invoke it.

Callers must supply expanded invoice lines. A loaded empty collection is valid;
unloaded lines and nil entries are critical validation issues. Discount
reconciliation checks live detailed lines against live amount-discount entries,
including rounding corrections, using billing's currency rounding. Deleted
parents, details and discount entries do not contribute to this check.

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
