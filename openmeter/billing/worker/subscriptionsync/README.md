# Subscription sync

Subscription sync is the reconciliation boundary between the desired schedule
owned by [Subscription](../../../subscription/README.md) and the billing-side
artifacts owned by [Billing](../../README.md) and
[Charges](../../charges/README.md).

It translates a subscription view into invoice lines or charge intents. It
does not edit subscriptions, calculate invoice totals, realize charges, or
advance invoice and payment lifecycles.

## Reconciliation model

A sync is given a subscription reference or expanded view and an `asOf`
horizon. It resolves the subscription, including deleted records, and the
customer's billing profile before acquiring billing's customer lock. Under that
lock it:

1. loads the persisted invoice-line and charge state owned by that subscription
2. builds the target billable periods from the resolved subscription view
3. diffs target and persisted items by their subscription-sync identity
4. applies invoice patches, then charge patches
5. records whether the subscription still has billables and when it next needs
   reconciliation

This is reconciliation, not event handling by side effect. Repeating the same
sync must converge. Subscription events provide the prompt path, invoice
creation triggers a refill of consumed future work, and the periodic reconciler
repairs missed or failed event processing.

Invoice patches, charge patches, and the sync-state update share the billing
transaction. A failure rolls back the plan rather than leaving only the earlier
patches applied.

The entrypoints that also invoice the customer run pending-line collection
after reconciliation so flat fees billed in advance can become invoices
immediately.

## Target time model

The target builder walks each phase and item version into service periods up to
the requested horizon. The horizon normally expands to the end of the aligned
billing period containing `asOf`; it is not simply a database query cutoff.

Each target item keeps three periods:

- service period: the portion of service represented by this artifact
- full service period: the untruncated period used for proration
- billing period: the subscription-aligned outer period

Flat fees billed in advance use the billing-period start as `InvoiceAt`.
Everything else is invoiced no earlier than both the service-period and
billing-period end.

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

Invoice-backed patches may create gathering lines or update mutable gathering
and standard invoices. If subscription state disagrees with an immutable
standard invoice, sync records validation issues; it does not rewrite the
invoice.

Charge-backed patches operate on the charge intent. Charge-managed invoice
lines are projections of charge state and are deliberately excluded from the
subscription-sync invoice read model. A customer-facing charge override also
does not erase the subscription-owned base intent: later subscription changes
continue to reconcile that base without resurrecting the overridden effective
charge.

## Deletion, cancellation, and retries

Cancellation syncs through the subscription end so artifacts are shortened or
removed according to the final desired periods. A deleted subscription has no
view and therefore produces an empty target, asking reconciliation to remove
its remaining owned artifacts subject to immutable-invoice rules. Cleanup must
use an ID-based entrypoint because the normal subscription view lookup excludes
deleted records.

Sync state is written only after the plan has been applied. `HasBillables`
prevents needless future scheduling; `NextSyncAfter` is the end of the
generated horizon. Errors remain retryable through the event handler or
periodic reconciler. Dry-run builds the same plan but only logs it and does not
write artifacts or sync state.

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
