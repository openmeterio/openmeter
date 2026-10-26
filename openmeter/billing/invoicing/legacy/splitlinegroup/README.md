# Legacy split-line groups

Split-line groups preserve a legacy progressive line's full service-period
identity while its children are materialized on different invoices. This package
owns group persistence, mutable group metadata, deletion guards, and the history
facts used by the legacy line engine and subscription reconciliation.
[Billing](../../../README.md) owns invoice and line identity, persistence,
lifecycle, and customer locks. Existing legacy artifacts retain their ownership;
group operations do not convert them into charges.

History reads load ordered, lightweight line and invoice facts, including deleted
children, rather than invoice aggregates. Previously billed amounts exclude
gathering children, deleted lines, deleted invoices (including terminal deleted
status without a deletion timestamp), and periods beyond the requested cutoff.
The line engine resolves complete history only for referenced groups and pairs
it with the authoritative pending line; ordinary invoice reads need no history.

Consumers use the group service; the adapter owns the database queries. Both
reuse the transaction carried by the caller. Standalone batch history reads
retain their existing read boundary without opening a transaction. Group creation during
splitting rolls back with failed invoice persistence. Subscription invoice
patches, subsequent group changes, and sync state commit or roll back together
under billing's customer lock. Repeated reads see invoice changes made in that
attempt; group history is not cached across writes.

Group updates validate mutable fields against the persisted price. Deletion
loads current children and rejects any live child; an already-marked-deleted
group retains the existing early-return behavior. Deleted children remain
available until group deletion. Group reads are scoped by namespace, including
their child lines and invoice headers.
