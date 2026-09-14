# Subscription-sync charge identity collision plan (v3)

Status: working plan. The long-term classifier is agreed in principle; a narrow delete/create fallback is proposed for the known impossible-shrink failure.

## Problem

Subscription sync correlates recurring billing artifacts by:

```text
{subscriptionID}/{phaseKey}/{itemKey}/v[{itemVersion}]/period[{periodIndex}]
```

The item-version component is a slice position. Unscheduling a future edit can remove a version, after which a different item can occupy the same position. The persisted charge and new target then have the same child unique reference even though they reference different physical subscription items and different base intents.

The current flow makes the collision harder to detect:

1. Load persisted charge state.
2. Build target state from the current subscription view.
3. `repairChargeSubscriptionReferences` matches by child unique reference and rewrites a mismatching `subscription_item_id`.
4. Reconciliation compares only `ServicePeriod.To` and chooses shrink or extend.

For the observed failure this turns:

```text
persisted charge: item A, $200, [Feb 1, Mar 1)
target charge:    item B, $300, [Jan 20, Feb 1)
```

into an attempted shrink of the old charge after its reference has already been changed from item A to item B. The resulting end is at or before the existing start, so charge validation rejects the patch and subscription sync remains retryable.

The local customer-workflow regressions are `TestUnschedulingAndReplacingFutureItemDoesNotReuseChargeIdentity` and `TestRecreatingFuturePhaseRepairsChargeSubscriptionReferences` in `openmeter/billing/worker/subscriptionsync/service/creditsonly_test.go`. They currently cover `credit_only`; a TODO records the missing `credit_then_invoice` coverage.

## Identity facts

- The charge already persists the physical subscription item ID in its subscription-owned base intent.
- The charge also persists the physical subscription phase ID. Like item rows, phase rows can be archived and recreated while their phase key and billing intent remain unchanged.
- A matching physical item ID is sufficient evidence that the target still belongs to the same materialized subscription item.
- A mismatching physical item ID is ambiguous because subscription materialization also recreates rows for harmless changes such as adjusting `active_to`.
- Customer overrides are not subscription source state. Subscription sync always compares and targets the persisted base intent, regardless of whether an override exists. It does not transfer, clear, or otherwise manage the override; the user remains responsible for the overridden charge.
- Realization and invoice history do not affect replacement classification. Subscription sync emits the normal system delete and leaves economic adjustment behavior to the charge domain. Deleted charges and their history remain queryable.
- Reference repair is valid only after proving that the different physical rows represent the same source intent.

No new stable subscription identifier is required for this approach.

## Intended classifier

For a target and persisted charge sharing the child unique reference:

| Physical subscription references | Base-intent semantics | Classification | Action |
| --- | --- | --- | --- |
| Item and phase IDs match | By contract unchanged | Same materialized item | No-op, shrink, or extend |
| Item or phase ID differs | Match after normalization | Harmless rematerialization | Repair both references, then no-op, shrink, or extend |
| Item or phase ID differs | Differ | Actual replacement in a reused slot | Delete the old charge, then create the target charge |

The second branch must continue into ordinary reconciliation after repairing the reference. The cadence change that caused rematerialization may itself require a shrink or extend patch.

### Semantic comparison

Construct the target charge intent with the existing flat-fee or usage-based create-intent mapper and compare it with the persisted charge's base intent.

Normalize out only representation and period-reconciliation differences:

- physical subscription item ID and physical subscription phase ID
- right-hand service, full-service, and billing-period boundaries
- `InvoiceAt` and values derived only from those right-hand boundaries
- charge ID, persistence timestamps, lifecycle status, realizations, resolved state, and validation issues

Keep source-definition fields in the comparison, including:

- charge type and settlement mode
- customer and subscription identity
- currency, feature, tax, and cost-basis intent
- price or amount before proration
- payment term, billing cadence, discounts, unit configuration, and proration configuration
- names, metadata, annotations, and other subscription-owned intent fields unless separately documented as non-semantic
- service, full-service, and billing-period start anchors

A matching item ID with a differing normalized base intent is an invariant violation. The initial implementation may report it rather than silently replacing the charge.

## Temporary fallback for the known impossible-shrink shape

The complete semantic matcher touches both charge types and several intent representations. To unblock the observed failures with a smaller change, detect the known impossible-shrink shape before `repairChargeSubscriptionReferences` runs.

The fallback candidate must satisfy all of the following:

- the persisted artifact is a subscription-managed charge
- target and persisted charge share the child unique reference
- the target subscription item ID differs from the charge base intent's item ID
- the target and existing service periods have different starts
- the target period ends at or before the existing period starts
- the concrete charge type is unchanged

For a candidate:

1. Do not repair the old charge's subscription item reference.
2. Add a system delete patch targeting the persisted charge's base intent through the collection selected from the existing artifact.
3. Add a create intent for the target through the collection selected from the target artifact.
4. Apply the delete and create in the existing transaction. Charge application already executes patches before creates so the reused child unique reference is released first.
5. Emit structured diagnostics identifying the fallback reason, unique reference, old and new item IDs, charge ID, and both service periods.

An active customer override does not change fallback eligibility or classification. It remains attached to the old charge and is neither inspected as source state nor transferred to the replacement charge.

Existing realization or invoice history also does not change fallback eligibility. Subscription sync does not calculate or select economic corrections; it issues the same system delete, and the owning charge lifecycle applies its normal deletion behavior. The deleted charge remains available for listing and audit.

This fallback intentionally handles only the observed impossible-shrink shape:

```text
target:   [earlier start, existing start]
existing: [existing start, later end]
```

It must not be described as general replacement detection. A differing item ID with overlapping periods remains outside the fallback until semantic matching is implemented. Such a case should fail explicitly and remain retryable rather than being silently repaired or replaced.

## Why delete then create

For the temporary fallback, the physical item IDs and incompatible periods prove that the target cannot be a shrink of the persisted charge. Treating the operation as replacement restores the state the system would have produced if billing had synchronized between the end-user operations:

- the removed future item would have caused deletion of its charge
- the later newly added item would have caused creation of a new charge

Delete/create preserves the old immutable source intent and any audit state under the old charge ID. The replacement receives a new charge ID and points directly to the target subscription item.

The fallback uses delete/create rather than an internal base-intent replacement because stable charge identity is not required to unblock the current failures. A future system-owned `ReplaceBaseIntent` operation remains possible if charge-ID stability becomes a product requirement.

## Reference-repair ordering

`repairChargeSubscriptionReferences` must no longer mutate every item-ID mismatch before planning.

The logical order becomes:

1. Build persisted and target state.
2. Pair items by child unique reference.
3. Compare physical subscription item IDs.
4. Classify a mismatch as temporary fallback, semantic replacement, or harmless rematerialization.
5. Repair references only for harmless rematerialization.
6. Plan no-op, shrink, extend, or delete/create.

The implementation may move repair into planning or pass the precomputed classification into the reconciler. It must not write the repaired reference until classification has completed.

## Test evidence

### Temporary fallback

- The existing end-user workflow regression completes successfully.
- The old `$200` charge is deleted and retains its original subscription item reference in history.
- A new `$300` charge is created with a different charge ID, the target item ID, target periods, and the reused child unique reference.
- Repeating subscription sync is a no-op.
- A mismatching item ID with an active override still targets the base intent; the override is left untouched and is not transferred to the replacement charge.
- Existing realization or invoice history does not change replacement classification; subscription sync emits the normal system delete and delegates its consequences to the charge lifecycle.
- A mismatching item ID with overlapping periods does not enter the temporary fallback.

### Intended classifier

- Matching item and phase IDs select ordinary no-op, shrink, and extend behavior.
- A differing item or phase ID with equal normalized base intents repairs both references and then selects ordinary period reconciliation.
- A differing item or phase ID with different normalized base intents selects delete/create.
- Matching item and phase IDs with different normalized base intents produce an integrity error.
- Flat-fee and usage-based comparisons cover both `credit_only` and `credit_then_invoice`.
- An active manual override never participates in base-intent comparison. Subscription sync targets the base intent without asking the user and leaves override management to the user.

## Implementation sequence

1. Extract item-ID access and replacement planning so classification happens before repair.
2. Implement the narrow impossible-shrink fallback and make the existing regression assert the complete delete/create outcome.
3. Add fallback guard and idempotency tests.
4. Add the normalized base-intent semantic matcher for flat-fee and usage-based charges.
5. Replace the temporary geometry heuristic with semantic classification.
6. Add phase-rematerialization and `credit_then_invoice` regression coverage, and verify that active overrides and economic history do not influence subscription-sync classification.
7. Update the subscription-sync and charge domain documentation with the final classifier and remove temporary-fallback language when the semantic matcher replaces it.

## Open decisions

- Whether annotations and metadata are always semantic for replacement detection or need a documented normalization rule.
