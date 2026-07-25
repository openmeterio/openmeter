package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
)

// repairChargeSubscriptionReferences repairs charge subscription item references before reconciliation.
//
// Subscription sync identifies billable items by logical path:
// subscription / phase key / item key / item version / billing period. That identity intentionally does
// not contain the concrete subscription_items.id, because most reconciliation decisions are about the
// billable path and period shape, not the DB row backing the current view.
//
// Subscription edits can still recreate the subscription item row for the same logical item. One known
// case is shrinking an item by inserting a later phase: the phase is preserved, but the item may be
// soft-deleted and recreated with the same key, same version, and a new active_to. In that case the target
// item and the persisted charge still share the same subscription-sync unique ID, so the reconciler should
// update/shrink the existing charge rather than delete and recreate it. However, the charge must continue
// to point at the concrete subscription item returned by the current subscription view.
//
// This repair is deliberately narrow: subscription_id and phase_id mismatches are treated as integrity
// errors, and only subscription_item_id is updated. The charge adapter keeps a matching TODO on the
// temporary mutability of subscription_item_id; it should become immutable again once subscription edits no
// longer recreate item IDs for logical item updates.
func (s *Service) repairChargeSubscriptionReferences(ctx context.Context, persisted persistedstate.State, target targetstate.State, dryRun bool) (persistedstate.State, error) {
	if s.chargesService == nil {
		return persisted, nil
	}

	billableTargetItems := lo.Filter(target.Items, func(item targetstate.StateItem, _ int) bool {
		return item.IsBillable()
	})
	billableTargetUniqueIDs := lo.Map(billableTargetItems, func(item targetstate.StateItem, _ int) string {
		return item.UniqueID
	})
	if len(lo.Uniq(billableTargetUniqueIDs)) != len(billableTargetUniqueIDs) {
		return persistedstate.State{}, fmt.Errorf("duplicate billable target unique IDs while repairing charge subscription references")
	}

	targetByUniqueID := lo.SliceToMap(billableTargetItems, func(item targetstate.StateItem) (string, targetstate.StateItem) {
		return item.UniqueID, item
	})

	for _, persistedEntry := range lo.Entries(persisted.ByUniqueID) {
		targetItem, ok := targetByUniqueID[persistedEntry.Key]
		if !ok {
			continue
		}

		chargeSubscription, err := persistedChargeSubscriptionReferenceFromItem(persistedEntry.Value)
		if err != nil {
			return persistedstate.State{}, fmt.Errorf("getting persisted charge subscription reference: %w", err)
		}
		if !chargeSubscription.IsCharge {
			continue
		}

		expectedSubscription := meta.SubscriptionReference{
			SubscriptionID: targetItem.Subscription.ID,
			PhaseID:        targetItem.PhaseID,
			ItemID:         targetItem.SubscriptionItem.ID,
		}

		if chargeSubscription.Subscription == nil {
			return persistedstate.State{}, fmt.Errorf("charge[%s] is missing subscription reference", chargeSubscription.ChargeID.ID)
		}

		if chargeSubscription.Subscription.SubscriptionID != expectedSubscription.SubscriptionID || chargeSubscription.Subscription.PhaseID != expectedSubscription.PhaseID {
			return persistedstate.State{}, fmt.Errorf("charge[%s] subscription reference mismatch: expected subscription[%s]/phase[%s], got subscription[%s]/phase[%s]",
				chargeSubscription.ChargeID.ID,
				expectedSubscription.SubscriptionID,
				expectedSubscription.PhaseID,
				chargeSubscription.Subscription.SubscriptionID,
				chargeSubscription.Subscription.PhaseID,
			)
		}

		if chargeSubscription.Subscription.ItemID == expectedSubscription.ItemID {
			continue
		}

		var updatedCharge charges.Charge
		if dryRun {
			updatedCharge, err = chargeSubscription.WithSubscriptionItemID(expectedSubscription.ItemID)
			if err != nil {
				return persistedstate.State{}, fmt.Errorf("updating charge subscription reference in memory: %w", err)
			}
		} else {
			// TODO: subscription edits can recreate subscription items while the
			// subscription-sync identity remains based on the logical path. Keep
			// charge references aligned here until subscription item identity or
			// target unique IDs can model this case directly.
			updatedCharge, err = chargeSubscription.UpdateSubscriptionItemID(ctx, s.chargesService, expectedSubscription.ItemID)
			if err != nil {
				return persistedstate.State{}, fmt.Errorf("updating charge subscription reference: %w", err)
			}
		}

		updatedPersistedItem, err := persistedItemFromCharge(updatedCharge)
		if err != nil {
			return persistedstate.State{}, fmt.Errorf("mapping updated charge to persisted item: %w", err)
		}

		persisted.ByUniqueID[persistedEntry.Key] = updatedPersistedItem
	}

	return persisted, nil
}

type persistedChargeSubscriptionReference struct {
	Charge       charges.Charge
	ChargeID     meta.ChargeID
	Subscription *meta.SubscriptionReference
	IsCharge     bool
}

func (r persistedChargeSubscriptionReference) UpdateSubscriptionItemID(ctx context.Context, chargesService charges.Service, newSubscriptionItemID string) (charges.Charge, error) {
	return chargesService.UpdateSubscriptionItemID(ctx, r.Charge, newSubscriptionItemID)
}

func (r persistedChargeSubscriptionReference) WithSubscriptionItemID(newSubscriptionItemID string) (charges.Charge, error) {
	if newSubscriptionItemID == "" {
		return charges.Charge{}, fmt.Errorf("subscription item ID is required")
	}

	switch r.Charge.Type() {
	case meta.ChargeTypeFlatFee:
		charge, err := r.Charge.AsFlatFeeCharge()
		if err != nil {
			return charges.Charge{}, err
		}

		baseIntent := charge.Intent.GetBaseIntent()
		subscription, err := subscriptionReferenceWithItemID(baseIntent.Subscription, newSubscriptionItemID)
		if err != nil {
			return charges.Charge{}, err
		}

		baseIntent.Subscription = subscription
		charge.Intent = flatfee.NewOverridableIntent(baseIntent, charge.Intent.GetOverrideLayerMutableFields())

		return charges.NewCharge(charge), nil
	case meta.ChargeTypeUsageBased:
		charge, err := r.Charge.AsUsageBasedCharge()
		if err != nil {
			return charges.Charge{}, err
		}

		baseIntent := charge.Intent.GetBaseIntent()
		subscription, err := subscriptionReferenceWithItemID(baseIntent.Subscription, newSubscriptionItemID)
		if err != nil {
			return charges.Charge{}, err
		}

		baseIntent.Subscription = subscription
		charge.Intent = usagebased.NewOverridableIntent(baseIntent, charge.Intent.GetOverrideLayerMutableFields())

		return charges.NewCharge(charge), nil
	default:
		return charges.Charge{}, fmt.Errorf("unsupported charge type: %s", r.Charge.Type())
	}
}

func subscriptionReferenceWithItemID(subscription *meta.SubscriptionReference, newSubscriptionItemID string) (*meta.SubscriptionReference, error) {
	if subscription == nil {
		return nil, fmt.Errorf("subscription reference is required")
	}

	subscriptionCopy := *subscription
	subscriptionCopy.ItemID = newSubscriptionItemID

	return &subscriptionCopy, nil
}

func persistedChargeSubscriptionReferenceFromItem(item persistedstate.Item) (persistedChargeSubscriptionReference, error) {
	switch item.Type() {
	case persistedstate.ItemTypeChargeFlatFee:
		charge, err := persistedstate.ItemAsFlatFeeCharge(item)
		if err != nil {
			return persistedChargeSubscriptionReference{}, err
		}

		return persistedChargeSubscriptionReference{
			Charge:       charges.NewCharge(charge),
			ChargeID:     charge.GetChargeID(),
			Subscription: charge.Intent.GetSubscription(),
			IsCharge:     true,
		}, nil
	case persistedstate.ItemTypeChargeUsageBased:
		charge, err := persistedstate.ItemAsUsageBasedCharge(item)
		if err != nil {
			return persistedChargeSubscriptionReference{}, err
		}

		return persistedChargeSubscriptionReference{
			Charge:       charges.NewCharge(charge),
			ChargeID:     charge.GetChargeID(),
			Subscription: charge.Intent.GetSubscription(),
			IsCharge:     true,
		}, nil
	default:
		return persistedChargeSubscriptionReference{}, nil
	}
}

func persistedItemFromCharge(charge charges.Charge) (persistedstate.Item, error) {
	switch charge.Type() {
	case meta.ChargeTypeFlatFee:
		flatFeeCharge, err := charge.AsFlatFeeCharge()
		if err != nil {
			return nil, err
		}

		return persistedstate.NewChargeItemFromChargeType(meta.ChargeTypeFlatFee, nil, &flatFeeCharge)
	case meta.ChargeTypeUsageBased:
		usageBasedCharge, err := charge.AsUsageBasedCharge()
		if err != nil {
			return nil, err
		}

		return persistedstate.NewChargeItemFromChargeType(meta.ChargeTypeUsageBased, &usageBasedCharge, nil)
	default:
		return nil, fmt.Errorf("unsupported charge type: %s", charge.Type())
	}
}
