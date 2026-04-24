package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
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

		// Resolve tax code on the spec's RateCard before deriving the entity input so
		// that the enrichment is applied to the source of truth (opts.itemSpec) rather
		// than relying on the implicit pointer sharing between opts.itemSpec.RateCard
		// and itemEntityInput.RateCard. If ToCreateSubscriptionItemEntityInput ever
		// clones the rate card, the current order would silently stop updating the spec.
		if err := s.resolveTaxCode(ctx, opts.sub.Namespace, opts.itemSpec.RateCard); err != nil {
			return res, fmt.Errorf("failed to resolve tax code: %w", err)
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

	switch {
	case meta.TaxConfig.Stripe != nil && meta.TaxConfig.Stripe.Code != "":
		// Existing path: resolve/create TaxCode from Stripe code.
		tc, err := s.TaxCode.GetOrCreateByAppMapping(ctx, taxcode.GetOrCreateByAppMappingInput{
			Namespace: namespace,
			AppType:   app.AppTypeStripe,
			TaxCode:   meta.TaxConfig.Stripe.Code,
		})
		if err != nil {
			return fmt.Errorf("failed to resolve tax code for stripe code %s: %w", meta.TaxConfig.Stripe.Code, err)
		}
		return rc.ChangeMeta(func(m productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
			m.TaxConfig.TaxCodeID = lo.ToPtr(tc.ID)
			return m, nil
		})

	case meta.TaxConfig.TaxCodeID != nil:
		// New path: caller supplied a taxCodeId — validate it exists and backfill app mappings.
		tc, err := s.TaxCode.GetTaxCode(ctx, taxcode.GetTaxCodeInput{
			NamespacedID: models.NamespacedID{
				Namespace: namespace,
				ID:        *meta.TaxConfig.TaxCodeID,
			},
		})
		if err != nil {
			if taxcode.IsTaxCodeNotFoundError(err) {
				return models.NewGenericValidationError(fmt.Errorf("tax code %s not found", *meta.TaxConfig.TaxCodeID))
			}
			return fmt.Errorf("failed to resolve tax code %s: %w", *meta.TaxConfig.TaxCodeID, err)
		}
		if m, ok := tc.GetAppMapping(app.AppTypeStripe); ok {
			return rc.ChangeMeta(func(meta productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
				if meta.TaxConfig.Stripe == nil {
					meta.TaxConfig.Stripe = &productcatalog.StripeTaxConfig{Code: m.TaxCode}
				}
				return meta, nil
			})
		}
		return nil

	default:
		return nil
	}
}
