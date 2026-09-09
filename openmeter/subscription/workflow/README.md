# Subscription migration

Migration applies a later version of the same catalog plan to an existing
subscription. The subscription ID, start, billing anchor, cancellation end,
phase timeline, settlement mode, and cost-basis policy remain intact. Migration
publishes an update event; it does not cancel the subscription or create a
replacement. General subscription edits still reject subscriptions with addons.

## Diff and effective time

The workflow builds a spec from the target plan using the subscription's
existing customer and timing inputs, then applies the existing addon purchases
and quantity schedule to that prospective spec. It compares this composed
offering against the current view by phase key and item key.

The [item diff](../patch/diff.go) compares rate cards, billing overrides, ownership, and
boolean-entitlement restoration counts over each item's timeline. Feature
reference expansion alone is not a change. For each key it finds the first
difference at or after the effective time:

- An unchanged schedule generates no patch. Existing item and entitlement IDs,
  item-version indexes, and service periods survive.
- A changed active item retains its historical version, ending at the first
  difference. The replacement occupies the next item-version index.
- An added item starts at the effective time or its later scheduled start.
- A removed item ends at the effective time; historical versions remain.
- Future versions on an affected key are replaced by the target schedule.
  Unchanged prefixes are retained, including an active prefix when only a
  future quantity segment differs.

The generated schedule patch is internal; it does not add a public edit API.
Unlike the single-version add/remove patches, it handles future addon quantity
segments and gaps without rewriting historical version indexes. For example,
with a current version ending January 20 and a future version starting then,
`PatchRemoveItem` on January 10 would end the **last (future)** version on
January 10; it would not truncate the active version or remove the future one.
Repeating remove does not help because it keeps targeting that last version.
The [schedule patch](../patch/itemschedule.go) retains the historical prefix
and replaces the affected suffix as one operation. Persistence,
entitlement scheduling, hooks, cost-basis resolution, and event publication
use the existing subscription update path.

The customer lock covers the migration read, diff, and update. The plan
reference advances in the same transaction as item materialization, with a
comparison against the previous plan ID. Updates recheck that reference after
acquiring the customer lock so an edit read before migration cannot overwrite
the new terms. `AdvancePlanReferenceInput.Validate` enforces a later version of
the same plan in both `validateSyncTarget` and the repository operation, even
when called outside the migration workflow. The repository also verifies the
target reference in the subscription namespace before advancing it.

## Addons

Purchases and their quantity histories remain attached to the same subscription.
Every nonzero quantity segment overlapping the post-migration subscription must
have a target-plan assignment, satisfy its phase restriction, and stay within
its quantity limit. Overlay incompatibilities also reject the operation.

The workflow diffs the composed target rather than restoring and rematerializing
the live subscription. This avoids merging or renumbering unaffected historical
addon versions. Changed versions carry the target base terms so a later addon
quantity change restores the migrated offering.

## Billing consequences

[Subscription sync](../../billing/worker/subscriptionsync/README.md) continues
to own billing artifacts. Retaining an item's logical path and service periods
preserves its billing reconciliation identity. Adding an XL compute rate card
therefore does not split the existing S/M/L periods or adjust their invoices.
The new item can produce its own normal charges or invoice lines.

Changing L's price interrupts L at the effective time even when the provider
considers the change beneficial or its calculated charge happens to be equal.
OpenMeter compares structure; deciding whether terms are non-adverse remains
the provider's responsibility.

## API and limits

Migration accepts immediate or next-billing-cycle timing under the running-edit
timing rules, including the restriction on crossing into another phase. With
next-cycle timing, the amended schedule and target plan reference are committed
now; affected item cadences take effect at the resolved boundary.

The existing `current` / `next` response envelope contains the before snapshot
and amended view of the same subscription. Clients must not interpret `next`
as a replacement subscription ID or `current` as a canceled subscription.

Migration compares the actual subscription offering against the target plan;
customer edits that differ from the target are replaced from the effective time.
It preserves phase metadata and rejects changes to phase keys/start times,
billing cadence, settlement mode, and proration configuration. Invoice-currency
and item-currency restrictions from ordinary updates still apply. `startingPhase`
is rejected; `billingAnchor` is deprecated and ignored for client compatibility. Use
subscription change for schedule resets or a different plan.

No database backfill or new schema is required. Existing subscriptions can use
this workflow. Public audit-history endpoints and special treatment of
economically beneficial changes are outside its scope.
