package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	addondiff "github.com/openmeterio/openmeter/openmeter/subscription/addon/diff"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func (s *service) AddAddon(ctx context.Context, subscriptionID models.NamespacedID, addonInp subscriptionworkflow.AddAddonWorkflowInput) (subscription.SubscriptionView, subscriptionaddon.SubscriptionAddon, error) {
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

		// Let's try to decode when the subscription should be patched
		if err := addonInp.Timing.ValidateForAction(subscription.SubscriptionActionChangeAddons, &subView); err != nil {
			return def, models.NewGenericValidationError(fmt.Errorf("invalid timing for adding add-on: %w", err))
		}

		editTime, err := addonInp.Timing.ResolveForSpec(spec)
		if err != nil {
			return def, fmt.Errorf("failed to resolve timing: %w", err)
		}

		if !subView.Subscription.IsActiveAt(editTime) {
			return def, models.NewGenericValidationError(fmt.Errorf("subscription is not active at the time of adding the addon"))
		}

		if len(subsAdds.Items) > 0 {
			// TODO: remove
			return def, models.NewGenericNotImplementedError(fmt.Errorf("adding addons to a subscription with existing addons is not supported"))
		}

		diffs, err := slicesx.MapWithErr(subsAdds.Items, func(subAdd subscriptionaddon.SubscriptionAddon) (addondiff.Diffable, error) {
			return addondiff.GetDiffableFromAddon(subView, subAdd)
		})
		if err != nil {
			return def, fmt.Errorf("failed to get diffable from addon: %w", err)
		}

		diffs = lo.Filter(diffs, func(diff addondiff.Diffable, _ int) bool {
			return diff != nil
		})
		if len(diffs) != len(subsAdds.Items) {
			return def, fmt.Errorf("failed to get diffable from addons, got %d addons but %d diffs", len(subsAdds.Items), len(diffs))
		}

		for _, diff := range diffs {
			if err := spec.Apply(diff.GetRestores(), subscription.ApplyContext{
				CurrentTime: editTime,
			}); err != nil {
				return def, fmt.Errorf("failed to restore subscription addon: %w", err)
			}
		}

		// Now let's try to purchase the addon

		subsAdd, err := s.AddonService.Create(ctx, subscriptionID.Namespace, subscriptionaddon.CreateSubscriptionAddonInput{
			MetadataModel:  addonInp.MetadataModel,
			AddonID:        addonInp.AddonID,
			SubscriptionID: subscriptionID.ID,
			InitialQuantity: subscriptionaddon.CreateSubscriptionAddonQuantityInput{
				ActiveFrom: editTime,
				Quantity:   addonInp.InitialQuantity,
			},
		})
		if err != nil {
			return def, fmt.Errorf("failed to create subscription addon: %w", err)
		}

		if subsAdd == nil {
			return def, errors.New("subscription addon is nil")
		}

		// Now let's reapply and sync
		for _, diff := range diffs {
			if err := spec.Apply(diff.GetApplies(), subscription.ApplyContext{
				CurrentTime: editTime,
			}); err != nil {
				return def, fmt.Errorf("failed to apply diff: %w", err)
			}
		}

		diff, err := addondiff.GetDiffableFromAddon(subView, *subsAdd)
		if err != nil {
			return def, fmt.Errorf("failed to get diffable from addon: %w", err)
		}

		if err := spec.Apply(diff.GetApplies(), subscription.ApplyContext{
			CurrentTime: editTime,
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
