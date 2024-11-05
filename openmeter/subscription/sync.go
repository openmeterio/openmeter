package subscription

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (c *commandAndQuery) Edit(ctx context.Context, subscriptionID models.NamespacedID, patches []Patch) (Subscription, error) {
	var def Subscription
	currentTime := clock.Now()

	// Fetch the subscription, check if it exists
	sub, err := c.Get(ctx, subscriptionID)
	if err != nil {
		return def, err
	}

	// Build the current spec from the Subscription
	spec, err := c.getSpec(ctx, sub)
	if err != nil {
		return def, fmt.Errorf("failed to get spec: %w", err)
	}

	// Check that all customizations are valid & set at times
	for i := range patches {
		if err := patches[i].Path().Validate(); err != nil {
			return def, err
		}

		p, err := SetAt(currentTime, patches[i])
		if err != nil {
			return def, err
		}
		patches[i] = p
	}

	err = spec.ApplyPatches(lo.Map(patches, ToApplies), ApplyContext{
		Operation: SpecOperationEdit,
	})
	if err != nil {
		return def, fmt.Errorf("failed to apply patches: %w", err)
	}

	view, err := c.Expand(ctx, subscriptionID)
	if err != nil {
		return def, fmt.Errorf("failed to expand: %w", err)
	}

	// We have to Validate that the the view's integrity is still intact
	// State diffs or system errors could produce invalid views
	err = view.Validate(true)
	if err != nil {
		return def, fmt.Errorf("failed to validate view, cannot execute edit: %w", err)
	}

	err = c.SyncByStateDiff(ctx, view, spec)
	if err != nil {
		return def, fmt.Errorf("failed to sync by state diff: %w", err)
	}

	// Once everything is successful, lets save the patches
	patchInputs, err := TransformPatchesForRepository(patches)
	if err != nil {
		return def, fmt.Errorf("failed to transform patches for repository: %w", err)
	}
	_, err = c.repo.CreateSubscriptionPatches(ctx, models.NamespacedID{
		ID:        sub.ID,
		Namespace: sub.Namespace,
	}, patchInputs)
	if err != nil {
		return def, fmt.Errorf("failed to create subscription patches: %w", err)
	}

	return c.Get(ctx, subscriptionID)
}

// FIXME: localize error so phase and item keys are always included (alongside subscription reference)
func (c *commandAndQuery) SyncByStateDiff(ctx context.Context, currentView SubscriptionView, newSpec *SubscriptionSpec) error {
	_, err := transaction.Run(ctx, c.transactionManager, func(ctx context.Context) (any, error) {
		if currentView == nil {
			// TODO: we can allow this, it should be the simple create case
			return nil, fmt.Errorf("current view is nil")
		}

		// First, lets check if something is different about the Subscription
		currentSpec := currentView.AsSpec()
		viewStruct, ok := currentView.(*subscriptionView)
		if !ok {
			return nil, fmt.Errorf("current view is not a subscriptionView")
		}
		_, err := c.subscriptionManager.SyncState(ctx, viewStruct, newSpec)
		if err != nil {
			return nil, err
		}

		// First, let's track everything in the current view (the previous state) while keeping track of what we touched
		currentPhases := currentView.Phases()
		for _, phase := range currentPhases {
			newPhaseSpec, ok := newSpec.Phases[phase.Key()]

			// If the phase is not present in the new spec, we should remove it
			if !ok {
				for _, item := range phase.Items() {
					// Manage Entitlement
					if ent, ok := item.Entitlement(); ok {
						_, err := c.entitlementManager.SyncState(ctx, &ent, nil)
						if err != nil {
							return nil, fmt.Errorf("failed to sync entitlement: %w", err)
						}
					}
					// Manage Price
					if price, ok := item.Price(); ok {
						_, err := c.priceManager.SyncState(ctx, &price, nil)
						if err != nil {
							return nil, fmt.Errorf("failed to sync price: %w", err)
						}
					}
				}
				continue
			}

			cadence, err := newSpec.GetPhaseCadence(newPhaseSpec.PhaseKey)
			if err != nil {
				return nil, fmt.Errorf("failed to get cadence for phase %s: %w", newPhaseSpec.PhaseKey, err)
			}

			// Phase doesn't exist as a resource, so we don't have to compare it...
			// If the phase is present in the new spec then we check it's items
			for _, item := range phase.Items() {
				// Lets check if the item exists
				newItemSpec, ok := newPhaseSpec.Items[item.Key()]

				// If it doesn't exist we should delete all it's resources
				if !ok {
					// Manage Entitlement
					if ent, ok := item.Entitlement(); ok {
						_, err := c.entitlementManager.SyncState(ctx, &ent, nil)
						if err != nil {
							return nil, fmt.Errorf("failed to sync entitlement: %w", err)
						}
					}
					// Manage Price
					if price, ok := item.Price(); ok {
						_, err := c.priceManager.SyncState(ctx, &price, nil)
						if err != nil {
							return nil, fmt.Errorf("failed to sync price: %w", err)
						}
					}
					continue
				}

				// If it does, then Item doesn't exist as a resource, so...
				// Lets check the entitlement
				ent, ok := item.Entitlement()
				var currentEntView *SubscriptionEntitlement
				if ok {
					currentEntView = &ent
				}
				var newEntSpec *SubscriptionEntitlementSpec
				if newItemSpec.HasEntitlement() {
					nSpec, err := newItemSpec.CreateEntitlementInput.ToSubscriptionEntitlementSpec(
						currentView.Sub().Namespace,
						currentView.Sub().ID,
						currentView.Customer().UsageAttribution.SubjectKeys[0],
						cadence,
						*newItemSpec,
					)
					if err != nil {
						return nil, fmt.Errorf("failed to calculate new entitlement spec for item %s in phase %s: %w", item.Key(), phase.Key(), err)
					}
					newEntSpec = nSpec
				}
				_, err := c.entitlementManager.SyncState(ctx, currentEntView, newEntSpec)
				if err != nil {
					return nil, fmt.Errorf("failed to sync entitlement: %w", err)
				}

				// Lets check the price
				pr, ok := item.Price()
				var currentPriceView *SubscriptionPrice
				if ok {
					currentPriceView = &pr
				}
				var newPriceSpec *CreatePriceSpec
				if newItemSpec.HasPrice() {
					s := newItemSpec.CreatePriceInput.ToCreatePriceSpec(
						currentView.Sub().Namespace,
						currentView.Sub().ID,
						cadence,
					)
					newPriceSpec = &s
				}
				_, err = c.priceManager.SyncState(ctx, currentPriceView, newPriceSpec)
				if err != nil {
					return nil, fmt.Errorf("failed to sync price: %w", err)
				}
			}
		}

		// Now we've validated that all resources previously present are in their new state.
		// Next, lets make sure that any new resources are created...
		for _, phase := range newSpec.GetSortedPhases() {
			// Let's check all it's items
			for _, item := range phase.Items {
				// If the item was already present then we can just skip it...
				if phase, ok := currentSpec.Phases[phase.PhaseKey]; ok {
					if _, ok := phase.Items[item.ItemKey]; ok {
						continue
					}
				}

				// If not, then we should create all the resources
				// Lets calculate the phase Cadence for the new spec
				cadence, err := newSpec.GetPhaseCadence(phase.PhaseKey)
				if err != nil {
					return nil, fmt.Errorf("failed to get cadence for phase %s: %w", phase.PhaseKey, err)
				}

				if item.HasEntitlement() {
					nSpec, err := item.CreateEntitlementInput.ToSubscriptionEntitlementSpec(
						currentView.Sub().Namespace,
						currentView.Sub().ID,
						currentView.Customer().UsageAttribution.SubjectKeys[0],
						cadence,
						*item,
					)
					if err != nil {
						return nil, fmt.Errorf("failed to calculate new entitlement spec for item %s in phase %s: %w", item.ItemKey, phase.PhaseKey, err)
					}
					_, err = c.entitlementManager.SyncState(ctx, nil, nSpec)
					if err != nil {
						return nil, fmt.Errorf("failed to sync entitlement: %w", err)
					}
				}

				if item.HasPrice() {
					nSpec := item.CreatePriceInput.ToCreatePriceSpec(
						currentView.Sub().Namespace,
						currentView.Sub().ID,
						cadence,
					)
					_, err := c.priceManager.SyncState(ctx, nil, &nSpec)
					if err != nil {
						return nil, fmt.Errorf("failed to sync price: %w", err)
					}
				}
			}
		}

		// Finally, everything should be in sync
		return nil, nil
	})

	return err
}
