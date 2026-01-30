package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *service) createPhase(
	ctx context.Context,
	cust customer.Customer,
	phaseSpec subscription.SubscriptionPhaseSpec,
	sub subscription.Subscription,
	cadence models.CadencedModel,
) (subscription.SubscriptionPhaseView, error) {
	return transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (subscription.SubscriptionPhaseView, error) {
		res := subscription.SubscriptionPhaseView{
			Spec:       phaseSpec,
			ItemsByKey: make(map[string][]subscription.SubscriptionItemView),
		}

		// First, let's create the phase itself
		phase, err := s.SubscriptionPhaseRepo.Create(ctx, phaseSpec.ToCreateSubscriptionPhaseEntityInput(sub, cadence.ActiveFrom))
		if err != nil {
			return res, fmt.Errorf("failed to create phase: %w", err)
		}

		res.SubscriptionPhase = phase

		// Second, let's create all items
		for key, itemSpecs := range phaseSpec.ItemsByKey {
			itemsByKey := make([]subscription.SubscriptionItemView, 0, len(itemSpecs))
			for _, itemSpec := range itemSpecs {
				item, err := s.createItem(ctx, createItemOptions{
					cust:         cust,
					sub:          sub,
					phase:        phase,
					phaseCadence: cadence,
					itemSpec:     *itemSpec,
				})
				if err != nil {
					return res, fmt.Errorf("failed to create item: %w", err)
				}

				if _, exists := res.ItemsByKey[item.SubscriptionItem.Key]; exists {
					return res, fmt.Errorf("item %s already exists", item.SubscriptionItem.Key)
				}

				itemsByKey = append(itemsByKey, item)
			}
			res.ItemsByKey[key] = itemsByKey
		}

		return res, nil
	})
}

type createItemOptions struct {
	cust         customer.Customer
	sub          subscription.Subscription
	phase        subscription.SubscriptionPhase
	phaseCadence models.CadencedModel
	itemSpec     subscription.SubscriptionItemSpec
}

func (s *service) createItem(
	ctx context.Context,
	opts createItemOptions,
) (subscription.SubscriptionItemView, error) {
	return transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (subscription.SubscriptionItemView, error) {
		res := subscription.SubscriptionItemView{
			Spec: opts.itemSpec,
		}

		itemCadence := opts.itemSpec.GetCadence(opts.phaseCadence)

		// First, let's see if we need to create an entitlement
		entInput, hasEnt, err := opts.itemSpec.ToScheduleSubscriptionEntitlementInput(
			subscription.ToScheduleSubscriptionEntitlementInputOptions{
				Customer:             opts.cust,
				Cadence:              itemCadence,
				PhaseStart:           opts.phaseCadence.ActiveFrom,
				AlignedBillingAnchor: opts.sub.BillingAnchor,
			},
		)
		if err != nil {
			return res, fmt.Errorf("failed to determine entitlement input for item %s: %w", opts.itemSpec.ItemKey, err)
		}

		var newEnt *entitlement.Entitlement

		if hasEnt {
			ent, err := s.EntitlementAdapter.ScheduleEntitlement(ctx, entInput, models.Annotations{
				subscription.AnnotationSubscriptionID: opts.sub.NamespacedID.ID,
			})
			if err != nil {
				return res, fmt.Errorf("failed to create entitlement: %w", err)
			}

			res.Entitlement = ent
			newEnt = &ent.Entitlement.Entitlement
		}

		// Second, let's create the item itself
		itemEntityInput, err := opts.itemSpec.ToCreateSubscriptionItemEntityInput(
			opts.phase.NamespacedID,
			opts.phaseCadence,
			newEnt,
		)
		if err != nil {
			return res, fmt.Errorf("failed to get item entity input: %w", err)
		}

		item, err := s.SubscriptionItemRepo.Create(ctx, itemEntityInput)
		if err != nil {
			return res, fmt.Errorf("failed to create item: %w", err)
		}

		res.SubscriptionItem = item

		return res, nil
	})
}

func (s *service) deletePhase(ctx context.Context, phase subscription.SubscriptionPhaseView) error {
	_, err := transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (any, error) {
		// To delete the phase, we need to delete all sub-resources of it.
		// Because deleting them is specific to the type of resource, we'll do it individually
		for _, items := range phase.ItemsByKey {
			for _, item := range items {
				if err := s.deleteItem(ctx, item); err != nil {
					return nil, fmt.Errorf("failed to delete item: %w", err)
				}
			}
		}

		// Let's delete the phase itself
		if err := s.SubscriptionPhaseRepo.Delete(ctx, phase.SubscriptionPhase.NamespacedID); err != nil {
			return nil, fmt.Errorf("failed to delete phase: %w", err)
		}

		return nil, nil
	})
	return err
}

func (s *service) deleteItem(ctx context.Context, item subscription.SubscriptionItemView) error {
	_, err := transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (any, error) {
		// If there's an entitlement let's delete it
		if item.Entitlement != nil {
			if err := s.EntitlementAdapter.DeleteByItemID(ctx, item.SubscriptionItem.NamespacedID); err != nil {
				return nil, fmt.Errorf("failed to delete entitlement: %w", err)
			}
		}

		// Let's delete the item itself
		if err := s.SubscriptionItemRepo.Delete(ctx, item.SubscriptionItem.NamespacedID); err != nil {
			return nil, fmt.Errorf("failed to delete item: %w", err)
		}

		return nil, nil
	})
	return err
}
