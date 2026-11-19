# Subscription sync

Subscription sync is the reconciliation boundary between the desired schedule
owned by [Subscription](../../../subscription/README.md) and the billing-side
artifacts owned by [Billing](../../README.md) and
[Charges](../../charges/README.md).

It translates a subscription view into invoice lines or charge intents. It
does not edit subscriptions, calculate invoice totals, realize charges, or
advance invoice and payment lifecycles.

## Reconciliation model

A sync is given a subscription ID and an `asOf` horizon. It first identifies the
customer, then acquires billing's customer lock and reloads the subscription and
billing profile. Unless deletion cleanup requires an empty target, it reloads the
current subscription view, including when triggered by an event carrying an older
view. Reading the view under this lock prevents a waiting sync from recreating
billing artifacts after another sync completed deletion cleanup. Under that lock it:

1. loads the persisted invoice-line and charge state owned by that subscription
2. builds the target billable periods from the resolved subscription view
3. diffs target and persisted items by their subscription-sync identity
4. applies invoice patches, then charge patches
5. records the sync outcome and, for a subscription with further billing work,
   when it next needs reconciliation

This is reconciliation, not event handling by side effect. Repeating the same
sync must converge. Subscription events provide the prompt path, invoice
creation triggers a refill of consumed future work, and the periodic reconciler
repairs missed or failed event processing.

Invoice patches, legacy split-group changes, charge patches, and the sync-state
update share the billing transaction. A failure rolls back the plan rather than leaving only the earlier
patches applied.

`SyncByID` collects the customer's pending lines after reconciliation when
`WithInvoicePendingLines()` is provided, so flat fees billed in advance can
become invoices immediately. Dry runs and deletion cleanup do not collect pending
lines.

## Target time model

The target builder walks each phase and item version into service periods up to
the requested horizon. `asOf` is not a cutoff: every sync provisions the aligned
billing period containing `asOf` and the one after it, across a phase change if
needed, so the following period's items, including usage billed in arrears,
exist before it begins. Service starting at the end of that following period is
left to the next sync, even an advance fee invoiceable at that boundary. Items
billed in arrears whose service period ends later still wait until they are
invoiceable. A subscription synced before its start, or before its first
billable phase, gets only that first billing period.

Each target item keeps three periods:

- service period: the portion of service represented by this artifact
- full service period: the untruncated period used for proration
- billing period: the subscription-aligned outer period

Flat fees billed in advance use the billing-period start as `InvoiceAt`.
Everything else is invoiced no earlier than both the service-period and
billing-period end.

A subscription with `BillingStartsAt` is not billed for service before that
time; see [Subscription](../../../subscription/README.md#cadence-and-effective-time).
The builder applies it in two steps.

First, the horizon is the later of `asOf` and the billing start. The first
billed billing period and the one after it therefore exist from the first sync,
and the generation limit and both sync deadlines follow from that horizon:
nothing is scheduled before billing starts. The horizon is not moved when the
subscription ends at or before the billing start, because then nothing is ever
billed.

Second, each generated item is matched against the billing start:

- service starting at or after it is billed unchanged
- service ending at or before it is dropped
- a recurring item spanning it is billed from the billing start, in the billing
  period that contains the billing start. Its full service period is kept, so
  the price is prorated when the subscription prorates prices. This happens
  when the billing start is inside a billing period, and for rate cards billed
  less often than the subscription
- a one-time fee that started earlier is dropped, even if its period ends later:
  it is charged for the moment the item starts, not for service over time

Billing periods never begin before the billing start, so fees billed in advance
are invoiced there at the earliest. Dropped periods keep their index: the first
billed artifact of a recurring item can be `period[3]`.

Shortening never leaves an empty period. Periods are compared at meter
resolution (whole seconds), so a billing period that ends within the second of
the billing start counts as over, and billing starts with the next one.

Recurring artifacts use the stable identity:

```text
{subscriptionID}/{phaseKey}/{itemKey}/v[{itemVersion}]/period[{periodIndex}]
```

One-time artifacts omit the period component. This identity is stored on both
invoice-backed and charge-backed state. Duplicate identities, including a
collision across the two backends, are an integrity error.

## Backend ownership

Persisted artifact type determines which backend continues to own an existing
item. New target items are routed according to the available charge stack,
feature gates, settlement mode, and price type. This permits gradual movement
from invoice-backed subscription items to charge-backed items without silently
replacing already persisted ownership.

Charge-backed reconciliation is the replacement path for supported new
artifacts. The invoice backend remains for compatibility while that transition
is incomplete. Existing artifacts keep their persisted backend rather than
being migrated implicitly.

The invoice backend loads group history and applies group changes through the
[legacy split-line group service](../../invoicing/legacy/splitlinegroup/README.md).
Invoice patches run before group updates, within the same customer lock and
transaction. Persisted group metadata therefore remains authoritative during
invoice calculation; reconciliation does not overlay pending group changes.

Invoice-backed patches may create gathering lines or update mutable gathering
and standard invoices. If subscription state disagrees with an immutable
standard invoice, sync records validation issues; it does not rewrite the
invoice.

Mutable legacy standard-line updates snapshot and rate through billing's
legacy engine before submission. Immutable comparisons retain snapshot-only
behavior; they do not change calculated invoice history. See
[Billing line ownership](../../README.md#line-ownership) for the calculation
and validation-issue contract.

The legacy engine resolves split-group context while invoice patches are applied.
Group period updates follow those invoice patches, so recalculation and immutable
comparisons still observe the persisted group period preceding the group update.

Charge-backed patches operate on the charge intent. Charge-managed invoice
lines are projections of charge state and are deliberately excluded from the
subscription-sync invoice read model. A customer-facing charge override also
does not erase the subscription-owned base intent: later subscription changes
continue to reconcile that base without resurrecting the overridden effective
charge. Credit-then-invoice usage reconciliation may preserve a missing feature
or meter as a product-catalog issue on a newly created charge; invoice assignment
keeps its gathering line pending until the dependency is repaired.

When a matched flat-fee or usage-based charge's physical phase or item reference
changes, sync compares the subscription-owned base intent's billing terms.
Currency identity, tax behavior and explicit tax-code ID, settlement mode,
feature key, cost basis, price, discounts, proration, and usage unit conversion
can distinguish a replacement charge. An omitted source tax-code ID accepts the
charge's stored ID; flat-fee payment terms are compared after applying the
catalog default. Presentation, annotations, discount lineage, expanded
currency data, pinned feature ID, plan attribution, periods, and invoice
scheduling do not distinguish billing terms. Historical plan and feature
attribution remain on the existing charge. Charge ownership and root
subscription identity are validated separately.

Equivalent terms permit repair of the physical reference, followed by normal
period reconciliation. A changed service-period end can still shrink or extend
the charge. An incompatible service-period start selects replacement because
shrink and extend only move the end. When physical references already match,
sync skips billing-term comparison and continues period reconciliation.

Before replacing a matched charge, sync loads all its realization runs, including
older and deleted runs, and checks every referenced invoice. An immutable
invoice blocks replacement, as does missing invoice history or a failed lookup.
This check runs only for the matched-charge replacement decision. Cancellation
deletes and ordinary period shrink or extend continue through their own paths.

This decision always follows system intent. Manual overrides remain associated
with the retired or repaired charge and do not participate in comparison; users
remain responsible for managing them. Charge deletion remains observable, while
economic corrections for already realized credits or invoice history remain the
responsibility of the downstream charge lifecycle rather than subscription sync.

## Deletion, cancellation, and retries

Cancellation syncs through the subscription end so artifacts are shortened or
removed according to the final desired periods. When a cancellation event's root
`UpdatedAt` predates the current subscription, the handler also uses the current
cancellation end as its horizon; a continued subscription needs no cancellation sync.
Annotation writes can also advance the root timestamp, so age alone must not
discard a still-valid cancellation. This timestamp is not a version for all child
changes. A deleted subscription has no view and
therefore produces an empty target, asking reconciliation to remove its remaining
owned artifacts subject to immutable-invoice rules. Cleanup must
load the deleted subscription record separately because the normal subscription
view lookup excludes deleted records.

A non-deleted subscription also has an empty target when its customer's
`DeletedAt` is at or before the subscription's `ActiveFrom`. That customer cannot
receive service from this subscription, but previously generated gathering lines
may still exist. Sync removes those artifacts instead of only stopping future
scheduling. It uses the deletion timestamp even when it is ahead of the worker
clock, so collection cannot invoice those lines while cleanup waits. Customer
deletion after subscription start still allows reconciliation of earlier service.
It caps recurring service periods at customer deletion even if the stored item
dates extend further. Full service periods remain unchanged for proration.

Sync state is written only after the plan has been applied. Errors remain
retryable through the event handler or periodic reconciler. Dry-run builds the
same plan but only logs it and does not write artifacts or sync state.

## Sync state

The sync-state table (`subscription_sync_states_v2`) keeps the last recorded
outcome and subscription revisions. Periodic reconciliation schedules from it; see
[Periodic reconciliation](#periodic-reconciliation). The table keeps its `_v2`
suffix from when it replaced the legacy `subscription_billing_sync_states` table,
which has since been dropped.

The states are:

- `new`: no outcome has been recorded; `last_success_at` is NULL.
- `active`: ordinary reconciliation succeeded. This also covers scheduled subscriptions with billables; it is not the subscription's lifecycle status.
- `no_billables`: reconciliation succeeded and no phase contains billable items. No next reconciliation is scheduled. Free phases followed by paid phases and one-time fees waiting for cancellation still have billables.
- `subscription_deleted`: an empty billing target was reconciled for a deleted subscription.
- `customer_deleted_before`: an empty target was reconciled because the customer was deleted at or before subscription start.
- `ended`: normal billing reconciliation succeeded at or after the effective end (the earlier of subscription `ActiveTo` and customer deletion), with a requested horizon covering that end.
- `too_old`: billing reconciliation was deliberately skipped because `ActiveTo` is more than 30 days before the worker clock. This outcome does not establish that billing artifacts were reconciled.

Deletion cleanup takes precedence over the age cutoff, with subscription deletion
checked before customer deletion. An age skip leaves billing artifacts unchanged,
skips pending-line collection, and records only the outcome without loading the
subscription view or billing artifacts.
Exactly 30 days old remains eligible for normal reconciliation.

Sync writes reconciliation outcomes in the same transaction as billing changes,
even when the plan was empty. Both a missing row and
`new` mean no outcome has been recorded. Other states require a non-NULL
`last_success_at`. Dry runs and failed syncs do not record outcomes.

An `active` result requires `next_reconciliation_at`, enforced by domain validation
and a PostgreSQL check. No other state may keep one, so `no_billables` and terminal
results clear a previous deadline. Sync records the result even when no billing
patches were needed, and only after applying billing changes: removing the last
billable item removes its pending artifacts before `no_billables` is recorded.

Sync picks the deadline from the last billing period it provisioned. This is the
period starting at the generation limit, which is normally the end of the billing
period containing `asOf`.

- If that period reaches the effective billing end (the earlier of `ActiveTo` and
  customer deletion), or generation already covers that end, the deadline is
  exactly that end, so the final sync can record `ended`.
- Otherwise the deadline is a random point 80–95% through the period.
- `ended` is recorded once both the worker clock and `AsOf` reach the effective end.
- A sync with a historical `AsOf` can leave a deadline in the past. That is
  intended: the subscription stays due until a sync with a current `AsOf` catches up.

Subscription create, update, cancel, continue, and delete hooks mark changes in
the subscription command's transaction. If no row exists, the hook inserts
`new` with no success timestamp and initializes both revisions to the same ULID.
Equal revisions on a `new` row do not mean it was reconciled. For an existing row,
the hook advances only `current_subs_revision` and `updated_at`, preserving the
previous outcome, success timestamp, and last synced revision. A failed marker
write rolls back the subscription command. Annotation-only writes do not mark
changes because root annotations do not affect subscription billing.

Each sync reads the sync state after acquiring the customer billing lock and
before reading the subscription view. `current_subs_revision` identifies the subscription
revision to reconcile; `last_synced_subs_revision` records the revision observed
at the start of the recorded sync outcome, including intentional age skips.
Both revisions are required on every row. For subscriptions without a row,
sync generates a ULID and recording the outcome inserts it into both columns.
On conflict, it updates the result, deadline, and last synced revision; the current
revision is never overwritten by sync. Every successful reconciliation records
an outcome, including an empty plan. Differing revisions make the subscription
due for periodic reconciliation; they never suppress a sync that was asked for.

## Periodic reconciliation

The periodic reconciler repairs what event handling missed and provisions the
next billing period on time. Each run selects candidates from two queries and
syncs them with the current time as `asOf`:

| Subscription | Sync state | Decision |
|---|---|---|
| deleted | no row, or state is not `subscription_deleted` | sync |
| deleted | `subscription_deleted` | skip |
| not deleted | no row | sync |
| not deleted | `new` | sync |
| not deleted | any other state, revisions differ | sync |
| not deleted | `active`, `next_reconciliation_at` ≤ now | sync |
| not deleted | anything else | skip |

The two candidate queries are the only place this rule lives:
`ListDeletedSubscriptionsToReconcile` applies the deleted rows and
`ListSubscriptionsToReconcile` the others. The reconciler syncs everything they
return, one page of candidates at a time, without evaluating the rule again. The
queries return the subscription and customer IDs: a sync needs only the
subscription, and the customer is there for workers that publish subscription
sync events, which are keyed by customer.

An event can sync a selected subscription before the periodic run reaches it.
The periodic sync is then skipped (`WithSkipIfSyncedSince`): the event may have
used a later horizon than the run would. A cancellation syncs up to the
subscription end, and a periodic sync as of now would remove the lines for the
billing periods after the next one. A forced run never skips.

A worker that publishes `SubscriptionSyncEvent`s instead of syncing directly sets
`SkipIfSyncedSince` on each event to the time it selected the candidates. The
event handler applies the same check, which also drops an event that is still
queued when a later one for the same subscription has already been handled.

A failed sync does not stop the run: the subscription stays due and the run
returns the error at its end. A canceled run stops before the next
subscription. Each run logs every page it starts and a summary when it ends.

The non-deleted query requires `DeletedAt` to be nil and the subscription to be
active at some point within the lookback before now. `ended`, `no_billables`,
`too_old` and `customer_deleted_before` subscriptions therefore drop out after one
sync, and come back only if they are edited. The cleanup query selects all
subscriptions with non-nil `DeletedAt` whose result is not `subscription_deleted`.
It has no service-period or deletion lookback: a missed deletion remains eligible
until cleanup succeeds.

Known limits of this selection:

- A subscription without a row predates sync-state tracking, so the first run
  after the switch from legacy scheduling syncs every such subscription in the lookback once.
- The lookback defaults to 30 days, the same age after which sync skips an ended
  subscription as `too_old`. An ended subscription whose final sync is missed for
  longer than the lookback is not listed again until the job runs with a wider
  lookback. An edit alone does not bring it back, because the lookback still
  excludes it, and by then a sync only records `too_old`, so repairing it also
  needs `--force-all`. A future-dated subscription is not listed before it starts.
- Deleting a customer does not change the subscription revision. An `active`
  subscription is reconciled at its next deadline rather than immediately. A
  `no_billables` subscription has no deadline and is not selected again until it
  is edited.
- A subscription created or edited just before a run still shows `new` or
  differing revisions while its event is in flight, so both can sync it. The
  customer billing lock serializes them. If the event gets there first the
  periodic sync is skipped; otherwise the event syncs it again. The skip compares
  the other sync's start time with the selection time, so it misses a sync that
  was already running when the candidates were selected, and workers whose
  clocks disagree. Both syncs then run: lines beyond the next billing period are
  removed and provisioned again as their deadlines come.
- The deadline lies in the last 5-20% of a billing period, so very short periods
  depend on how often the job runs.

The CLI's `--f` flag syncs every subscription in the lookback regardless of its
sync state. `--force-all` also enables `ForceResyncOldSubscriptions` to reconcile
subscriptions older than the 30-day cutoff. The `--l` lookback still limits
selection and defaults to 30 days; resyncing older history also requires a wider
lookback. Forced reconciliation still excludes deleted subscriptions with a
successful cleanup result. To put one back in the periodic cleanup queue, delete
its `subscription_sync_states_v2` entry. `list` prints what a run would sync;
`list --f` prints what a forced run would select. With a lookback wider than 30
days that includes subscriptions `all --f` only records as `too_old`;
reconciling those needs `all --force-all`.

## Race between billing collection and subscription deletion cleanup

Only scheduled subscriptions can be deleted, but subscription sync may already
have generated gathering lines for their future service periods. Deleting the
subscription does not synchronously remove those lines. Billing collection can
pick them up while subscription sync is still waiting to reconcile the deletion.
Cleanup must therefore remove them as soon as it observes a deletion marker,
without waiting for the subscription to start or for `DeletedAt` to pass.

The subscription's inherited `IsDeleted()` and `IsDeletedAt(t)` accessors answer
whether `DeletedAt` has passed at the supplied time. Subscription deletion writes
the deleting process's current time; it has no supported delayed-delete operation.
If that process's clock is ahead of the sync worker's clock, these accessors can
still return false for a subscription whose deletion has already committed.
**Subscription reconciliation intentionally uses `DeletedAt != nil` instead.**
Do not replace this check with the timestamp-based accessors.

For example, with `DeletedAt = worker now + 2 seconds`, using `IsDeleted()` would
let sync load the subscription view and preserve or generate gathering lines
instead of removing them. Billing collection could then invoice those lines.
The regular candidate list must exclude this subscription, the cleanup query
must include it immediately, and sync must use an empty target and skip pending-line
invoicing. All three decisions must use the same non-nil deletion check. The
[future-timestamp cleanup regression test](service/sync_test.go) verifies that
cleanup runs before `DeletedAt` and that completion removes the subscription
from subsequent polls.

Sync reloads the subscription after acquiring the customer billing lock so a
waiting sync cannot recreate lines from a view read before deletion cleanup.
Cleanup and its completion record share that transaction. Existing
immutable-invoice rules still apply if billing collection finishes first.

## Intentional limitations

- invoice-backed subscription artifacts remain fiat-only; custom-currency items
  require the charges backend
- charge intents preserve each subscription item's fiat or managed custom
  currency. For `credit_then_invoice`, subscription cost-basis mode maps to a
  dynamic charge cost basis or the subscription's pinned cost-basis resource;
  the charge lifecycle performs overage conversion into invoice currency
- subscription-owned credit-purchase charges are unsupported
- immutable invoice drift is reported, not automatically corrected
- an `asOf` at the current instant is not a request to provision the entire
  future subscription; callers that need future artifacts must provide the
  corresponding horizon
