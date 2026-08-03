package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *service) createPhaseWithChildren(
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
		phase, err := s.createPhase(ctx, phaseSpec, sub, cadence)
		if err != nil {
			return res, fmt.Errorf("failed to create phase: %w", err)
		}

		res.SubscriptionPhase = phase

		// Second, let's create all items
		for key, itemSpecs := range phaseSpec.ItemsByKey {
			itemsByKey := make([]subscription.SubscriptionItemView, 0, len(itemSpecs))
			for _, itemSpec := range itemSpecs {
				item, err := s.createItemWithEntitlement(ctx, createItemOptions{
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

func (s *service) createPhase(
	ctx context.Context,
	phaseSpec subscription.SubscriptionPhaseSpec,
	sub subscription.Subscription,
	cadence models.CadencedModel,
) (subscription.SubscriptionPhase, error) {
	return s.SubscriptionPhaseRepo.Create(ctx, phaseSpec.ToCreateSubscriptionPhaseEntityInput(sub, cadence.ActiveFrom))
}

type createItemOptions struct {
	cust         customer.Customer
	sub          subscription.Subscription
	phase        subscription.SubscriptionPhase
	phaseCadence models.CadencedModel
	itemSpec     subscription.SubscriptionItemSpec
}

func (s *service) createItemWithEntitlement(
	ctx context.Context,
	opts createItemOptions,
) (subscription.SubscriptionItemView, error) {
	return transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (subscription.SubscriptionItemView, error) {
		res := subscription.SubscriptionItemView{
			Spec: opts.itemSpec,
		}

		// First, let's see if we need to create an entitlement
		entInput, hasEnt, err := getItemEntitlementInput(opts)
		if err != nil {
			return res, fmt.Errorf("failed to determine entitlement input for item %s: %w", opts.itemSpec.ItemKey, err)
		}

		var newEnt *subscription.SubscriptionEntitlement

		if hasEnt {
			ent, err := s.createItemEntitlement(ctx, opts.sub, entInput)
			if err != nil {
				return res, fmt.Errorf("failed to create entitlement: %w", err)
			}

			res.Entitlement = ent
			newEnt = ent
		}

		// Second, let's create the item itself
		item, err := s.createItem(ctx, opts, newEnt)
		if err != nil {
			return res, fmt.Errorf("failed to create item: %w", err)
		}

		res.SubscriptionItem = item

		return res, nil
	})
}

func getItemEntitlementInput(opts createItemOptions) (subscription.ScheduleSubscriptionEntitlementInput, bool, error) {
	return opts.itemSpec.ToScheduleSubscriptionEntitlementInput(
		subscription.ToScheduleSubscriptionEntitlementInputOptions{
			Customer:             opts.cust,
			Cadence:              opts.itemSpec.GetCadence(opts.phaseCadence),
			PhaseStart:           opts.phaseCadence.ActiveFrom,
			AlignedBillingAnchor: opts.sub.BillingAnchor,
		},
	)
}

func (s *service) createItemEntitlement(
	ctx context.Context,
	sub subscription.Subscription,
	input subscription.ScheduleSubscriptionEntitlementInput,
) (*subscription.SubscriptionEntitlement, error) {
	return s.EntitlementAdapter.ScheduleEntitlement(ctx, input, models.Annotations{
		subscription.AnnotationSubscriptionID: sub.NamespacedID.ID,
	})
}

func (s *service) createItem(
	ctx context.Context,
	opts createItemOptions,
	ent *subscription.SubscriptionEntitlement,
) (subscription.SubscriptionItem, error) {
	// Resolve tax code on the spec's RateCard before deriving the entity input so
	// that the enrichment is applied to the source of truth (opts.itemSpec) rather
	// than relying on the implicit pointer sharing between opts.itemSpec.RateCard
	// and itemEntityInput.RateCard. If ToCreateSubscriptionItemEntityInput ever
	// clones the rate card, the current order would silently stop updating the spec.
	if err := s.resolveTaxCode(ctx, opts.sub.Namespace, opts.itemSpec.RateCard); err != nil {
		return subscription.SubscriptionItem{}, fmt.Errorf("failed to resolve tax code: %w", err)
	}

	itemEntityInput, err := opts.itemSpec.ToCreateSubscriptionItemEntityInput(
		opts.phase.NamespacedID,
		opts.phaseCadence,
		convert.SafeDeRef(ent, func(s subscription.SubscriptionEntitlement) *entitlement.Entitlement {
			return &s.Entitlement.Entitlement
		}),
	)
	if err != nil {
		return subscription.SubscriptionItem{}, fmt.Errorf("failed to get item entity input: %w", err)
	}

	return s.SubscriptionItemRepo.Create(ctx, itemEntityInput)
}

func (s *service) deletePhaseWithChildren(ctx context.Context, phase subscription.SubscriptionPhaseView) error {
	_, err := transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (any, error) {
		// To delete the phase, we need to delete all sub-resources of it.
		// Because deleting them is specific to the type of resource, we'll do it individually
		for _, items := range phase.ItemsByKey {
			for _, item := range items {
				if err := s.deleteItemWithEntitlement(ctx, item); err != nil {
					return nil, fmt.Errorf("failed to delete item: %w", err)
				}
			}
		}

		// Let's delete the phase itself
		if err := s.deletePhase(ctx, phase.SubscriptionPhase); err != nil {
			return nil, fmt.Errorf("failed to delete phase: %w", err)
		}

		return nil, nil
	})
	return err
}

func (s *service) deletePhase(ctx context.Context, phase subscription.SubscriptionPhase) error {
	return s.SubscriptionPhaseRepo.Delete(ctx, phase.NamespacedID)
}

func (s *service) deleteItemWithEntitlement(ctx context.Context, item subscription.SubscriptionItemView) error {
	_, err := transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (any, error) {
		// If there's an entitlement let's delete it
		if item.Entitlement != nil {
			if err := s.deleteItemEntitlement(ctx, item.SubscriptionItem); err != nil {
				return nil, fmt.Errorf("failed to delete entitlement: %w", err)
			}
		}

		// Let's delete the item itself
		if err := s.deleteItem(ctx, item.SubscriptionItem); err != nil {
			return nil, fmt.Errorf("failed to delete item: %w", err)
		}

		return nil, nil
	})
	return err
}

func (s *service) deleteItemEntitlement(ctx context.Context, item subscription.SubscriptionItem) error {
	return s.EntitlementAdapter.DeleteByItemID(ctx, item.NamespacedID)
}

func (s *service) deleteItem(ctx context.Context, item subscription.SubscriptionItem) error {
	return s.SubscriptionItemRepo.Delete(ctx, item.NamespacedID)
}

// resolveTaxCode ensures that a RateCard with a Stripe tax code in its TaxConfig
// has a corresponding TaxCode entity in the namespace. If no matching TaxCode exists,
// one is created. The RateCard's TaxConfig.TaxCodeID is then populated.
func (s *service) resolveTaxCode(ctx context.Context, namespace string, rc productcatalog.RateCard) error {
	if s.TaxCode == nil {
		return nil
	}

	meta := rc.AsMeta()
	if meta.TaxConfig == nil {
		return nil
	}

	return rc.ChangeMeta(func(m productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
		if err := productcatalog.ResolveTaxConfig(ctx, s.TaxCode, namespace, m.TaxConfig); err != nil {
			return m, err
		}
		return m, nil
	})
}
