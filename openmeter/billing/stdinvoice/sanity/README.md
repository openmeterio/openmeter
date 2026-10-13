# Invoice sanity checks

Sanity validation checks whether calculated invoice totals agree with the detail
consumed by invoicing apps. It is read-only and does not rate charges, repair
missing data, or establish that the commercial intent was correct.

Callers must supply expanded invoice lines. A loaded empty collection is valid;
unloaded lines and nil entries are critical validation issues. Discount
reconciliation checks live detailed lines against live amount-discount entries,
including rounding corrections, using billing's currency rounding. Deleted
parents, details and discount entries do not contribute to this check.

Failures use the `billing.invoice.sanity` component and preserve individual issue
codes, field paths and reconciliation values. The checker is separate from
ordinary model validation so incomplete historical records remain readable.
The invoice state machine checks before invoicing app calls for validation,
synchronization, finalization, optional post-advance hooks and asynchronous
readiness checks. Deletion and post-advance hooks in deletion states bypass
sanity validation so inconsistent invoices can still be cleaned up. Issuing
checks again after line finalization and before each external call, including
retries.

`billing.enforceInvoiceSanityChecks` defaults to `false`. Failures are always
logged at error severity. In log-only mode the app call proceeds without adding
blocking issues. Enforcement skips the app call and records critical sanity
issues through the existing lifecycle failure handling. A successful check or
disabled enforcement clears stale sanity issues when the operation is attempted
again, preserving unrelated issues.

Readiness probes log failures and withhold the app call when enforced, but never
persist issues or change state. A rejected readiness check leaves the invoice
in its current syncing state. Sanity checks run at app call sites; advancement
does not perform a separate check.
