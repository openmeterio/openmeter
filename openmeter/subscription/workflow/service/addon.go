package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	addondiff "github.com/openmeterio/openmeter/openmeter/subscription/addon/diff"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func (s *service) AddAddon(ctx context.Context, subscriptionID models.NamespacedID, addonInp subscriptionaddon.CreateSubscriptionAddonInput) (subscription.SubscriptionView, subscriptionaddon.SubscriptionAddon, error) {
	var def1 subscription.SubscriptionView
	var def2 subscriptionaddon.SubscriptionAddon

	if err := addonInp.Validate(); err != nil {
		return def1, def2, models.NewGenericValidationError(err)
	}

	// TODO: maybe we should lock the subscription for this operation
	res, err := transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (purchaseRes, error) {
		var def purchaseRes

		subView, err := s.Service.GetView(ctx, subscriptionID)
		if err != nil {
			return def, fmt.Errorf("failed to get subscription: %w", err)
		}

		subsAdds, err := s.AddonService.List(ctx, subscriptionID.Namespace, subscriptionaddon.ListSubscriptionAddonsInput{
			SubscriptionID: subscriptionID.ID,
		})
		if err != nil {
			return def, fmt.Errorf("failed to list subscription addons: %w", err)
		}

		if lo.SomeBy(subsAdds.Items, func(subAdd subscriptionaddon.SubscriptionAddon) bool {
			return subAdd.Addon.ID == addonInp.AddonID
		}) {
			return def, models.NewGenericConflictError(fmt.Errorf("subscription already has that addon purchased"))
		}

		// Let's get a clean spec by restoring the subscription
		spec := subView.AsSpec()

		calcTime := clock.Now()

		diffs, err := slicesx.MapWithErr(subsAdds.Items, func(subAdd subscriptionaddon.SubscriptionAddon) (addondiff.Diffable, error) {
			return addondiff.GetDiffableFromAddon(subView, subAdd)
		})
		if err != nil {
			return def, fmt.Errorf("failed to get diffable from addon: %w", err)
		}

		for _, diff := range diffs {
			if err := spec.Apply(diff.GetRestores(), subscription.ApplyContext{
				CurrentTime: calcTime,
			}); err != nil {
				return def, fmt.Errorf("failed to restore subscription addon: %w", err)
			}
		}

		// Now let's try to purchase the addon
		subsAdd, err := s.AddonService.Create(ctx, subscriptionID.Namespace, addonInp)
		if err != nil {
			return def, fmt.Errorf("failed to create subscription addon: %w", err)
		}

		if subsAdd == nil {
			return def, errors.New("subscription addon is nil")
		}

		// Now let's reapply and sync
		for _, diff := range diffs {
			if err := spec.Apply(diff.GetApplies(), subscription.ApplyContext{
				CurrentTime: calcTime,
			}); err != nil {
				return def, fmt.Errorf("failed to apply diff: %w", err)
			}
		}

		diff, err := addondiff.GetDiffableFromAddon(subView, *subsAdd)
		if err != nil {
			return def, fmt.Errorf("failed to get diffable from addon: %w", err)
		}

		if err := spec.Apply(diff.GetApplies(), subscription.ApplyContext{
			CurrentTime: calcTime,
		}); err != nil {
			return def, fmt.Errorf("failed to apply diff: %w", err)
		}

		_, err = s.Service.Update(ctx, subscriptionID, spec)
		if err != nil {
			return def, fmt.Errorf("failed to update subscription: %w", err)
		}

		subView, err = s.Service.GetView(ctx, subscriptionID)
		if err != nil {
			return def, fmt.Errorf("failed to get subscription: %w", err)
		}

		return purchaseRes{
			sub:    subView,
			subAdd: *subsAdd,
		}, nil
	})
	if err != nil {
		return def1, def2, err
	}

	return res.sub, res.subAdd, nil
}

type purchaseRes struct {
	sub    subscription.SubscriptionView
	subAdd subscriptionaddon.SubscriptionAddon
}

// The sub has addons if it has a non-0 quantity on any of them during its cadence
func hasAddons(view subscription.SubscriptionView, addons []subscriptionaddon.SubscriptionAddon) bool {
	subPer := view.Subscription.CadencedModel.AsPeriod()

	for _, add := range addons {
		for _, addInst := range add.GetInstances() {
			if addInst.Quantity > 0 {
				if addInst.CadencedModel.AsPeriod().Intersection(subPer) != nil {
					return true
				}
			}
		}
	}

	return false
}
