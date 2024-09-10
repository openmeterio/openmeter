package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *service) Update(ctx context.Context, subscriptionID models.NamespacedID, newSpec subscription.SubscriptionSpec) (subscription.Subscription, error) {
	var def subscription.Subscription

	// Get the full view
	view, err := s.GetView(ctx, subscriptionID)
	if err != nil {
		return def, fmt.Errorf("failed to get view: %w", err)
	}

	// Let's make sure edit is possible based on the transition rules
	if err := subscription.NewStateMachine(
		view.Subscription.GetStatusAt(clock.Now()),
	).CanTransitionOrErr(ctx, subscription.SubscriptionActionUpdate); err != nil {
		return def, err
	}

	return s.sync(ctx, view, newSpec)
}

// TODO: localize error so phase and item keys are always included (alongside subscription reference)
func (s *service) sync(ctx context.Context, view subscription.SubscriptionView, newSpec subscription.SubscriptionSpec) (subscription.Subscription, error) {
	return transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (subscription.Subscription, error) {
		var def subscription.Subscription

		// Some sanity checks for good measure
		if view.Subscription.CustomerId != newSpec.CustomerId {
			return def, fmt.Errorf("cannot change customer id")
		}
		if view.Subscription.Plan.Key != newSpec.Plan.Key {
			return def, fmt.Errorf("cannot change plan key")
		}
		if view.Subscription.Plan.Version != newSpec.Plan.Version {
			return def, fmt.Errorf("cannot change plan version")
		}
		if !view.Subscription.ActiveFrom.Equal(newSpec.ActiveFrom) {
			return def, fmt.Errorf("cannot change subscription active from")
		}

		// 1. Subscription Cadence has to match
		if !view.Subscription.CadencedModel.Equal(models.CadencedModel{ActiveFrom: newSpec.ActiveFrom, ActiveTo: newSpec.ActiveTo}) {
			_, err := s.SubscriptionRepo.SetEndOfCadence(ctx, view.Subscription.NamespacedID, newSpec.ActiveTo)
			if err != nil {
				return def, fmt.Errorf("failed to set end of cadence: %w", err)
			}
		}

		// 2. Anything that's changed or was removed has to be updated
		newSortedPhaseSpecs := newSpec.GetSortedPhases()
		for _, currentPhaseView := range view.Phases {
			// Let's try find a matching phase in the new spec
			matchingPhaseFromNewSpec, found := lo.Find(newSortedPhaseSpecs, func(s *subscription.SubscriptionPhaseSpec) bool {
				return s.PhaseKey == currentPhaseView.SubscriptionPhase.Key
			})

			// If there's no equivalent found in the current spec, we need to delete the phase
			if !found {
				if err := s.deletePhase(ctx, currentPhaseView); err != nil {
					return def, fmt.Errorf("failed to delete phase: %w", err)
				}

				// There's nothing more to be done for this phase, so lets skip to the next one
				continue
			}

			// sanity check
			if matchingPhaseFromNewSpec == nil {
				return def, fmt.Errorf("failed to find matching phase in new spec but no error was returned")
			}

			// Let's get the cadence of the current phase
			cadenceOfCurrentPhaseBasedOnSpec, err := view.Spec.GetPhaseCadence(currentPhaseView.SubscriptionPhase.Key)
			if err != nil {
				return def, fmt.Errorf("failed to get cadence for current phase %s: %w", currentPhaseView.SubscriptionPhase.Key, err)
			}

			// Let's get the cadence of the new phase
			cadenceOfNewPhaseBasedOnSpec, err := newSpec.GetPhaseCadence(matchingPhaseFromNewSpec.PhaseKey)
			if err != nil {
				return def, fmt.Errorf("failed to get cadence for new phase %s: %w", matchingPhaseFromNewSpec.PhaseKey, err)
			}

			// Lets figure out when the new phase should start
			newPhaseStartTime, _ := matchingPhaseFromNewSpec.StartAfter.AddTo(view.Subscription.ActiveFrom)

			curr := currentPhaseView.Spec.ToCreateSubscriptionPhaseEntityInput(view.Subscription, currentPhaseView.SubscriptionPhase.ActiveFrom)
			new := matchingPhaseFromNewSpec.ToCreateSubscriptionPhaseEntityInput(view.Subscription, newPhaseStartTime)

			currentPhaseIntact := true
			phaseToLinkToForNewResources := currentPhaseView.SubscriptionPhase

			// If the phase has any changes, we need to recreate it. That means also all sub-resources of it have to be relinked.
			if !curr.Equal(new) {
				// This means deleting the phase with all its sub-resources
				if err := s.deletePhase(ctx, currentPhaseView); err != nil {
					return def, fmt.Errorf("failed to delete phase: %w", err)
				}

				currentPhaseIntact = false
			}

			if !currentPhaseIntact {
				// Then we also need to re-create the phase
				newPhase, err := s.SubscriptionPhaseRepo.Create(ctx, new)
				if err != nil {
					return def, fmt.Errorf("failed to create phase: %w", err)
				}
				phaseToLinkToForNewResources = newPhase
			}

			// Now let's iterate through all items in the phase
			for currentItemViewsKey, currentItemViews := range currentPhaseView.ItemsByKey {
				// Let's try find a matching item in the new spec
				// Here as we do an update, we rely on the previously verified integrity of both view and spec
				// Due to this, we use a simple matching based on the index of the item under the given key
				matchingItemsByKeyFromNewSpec, found := matchingPhaseFromNewSpec.ItemsByKey[currentItemViewsKey]

				// If no matching key is found in the new spec lets delete everything
				if !found || len(matchingItemsByKeyFromNewSpec) == 0 {
					for _, currentItemView := range currentItemViews {
						if currentPhaseIntact {
							if err := s.deleteItem(ctx, currentItemView); err != nil {
								return def, fmt.Errorf("failed to delete item: %w", err)
							}
						}
					}

					// There's nothing more to be done for this item, so lets skip to the next one
					continue
				}

				for currentItemIdx, currentItemView := range currentItemViews {
					// Let's get the item with the same index from the new spec
					if currentItemIdx >= len(matchingItemsByKeyFromNewSpec) {
						// Let's delete the item as it's not present in the new spec

						if err := s.deleteItem(ctx, currentItemView); err != nil {
							return def, fmt.Errorf("failed to delete item: %w", err)
						}

						// There's nothing more to be done for this item, so lets skip to the next one
						continue
					}

					matchingItemFromNewSpec := matchingItemsByKeyFromNewSpec[currentItemIdx]

					currentItemIntact := currentPhaseIntact

					// First, let's check if the entitlemnet needs to be changed. The entitlement needs to change if:
					// 1. The item itself changes
					// 2. Or the create input based on the entitlement changes.

					// First, let's check if the item itself needs to be changed
					curr, err := currentItemView.Spec.ToCreateSubscriptionItemEntityInput(
						currentPhaseView.SubscriptionPhase,
						cadenceOfCurrentPhaseBasedOnSpec,
						convert.SafeDeRef(currentItemView.Entitlement, func(s subscription.SubscriptionEntitlement) *entitlement.Entitlement {
							return &s.Entitlement
						}),
					)
					if err != nil {
						return def, fmt.Errorf("failed to convert item to entity input: %w", err)
					}

					// Let's try to figure out what the cadence of new items would be
					cadenceOfThisPhaseBasedOnNewSpec, err := newSpec.GetPhaseCadence(phaseToLinkToForNewResources.Key)
					if err != nil {
						return def, fmt.Errorf("failed to get cadence for phase %s: %w", phaseToLinkToForNewResources.Key, err)
					}

					cadenceForItem, err := matchingItemFromNewSpec.GetCadence(cadenceOfThisPhaseBasedOnNewSpec)
					if err != nil {
						return def, fmt.Errorf("failed to get cadence for item %s: %w", matchingItemFromNewSpec.ItemKey, err)
					}

					newOnlyForComparisonWithInvalidEntitlement, err := matchingItemFromNewSpec.ToCreateSubscriptionItemEntityInput(
						phaseToLinkToForNewResources,
						cadenceOfNewPhaseBasedOnSpec,
						// Without the "new" entitlement already present we cannot properly compare the two create inputs.
						// To work around this, we'll reuse the current entitlement for comparison.
						// FIXME: This is a lie
						convert.SafeDeRef(currentItemView.Entitlement, func(s subscription.SubscriptionEntitlement) *entitlement.Entitlement {
							return &s.Entitlement
						}),
					)
					if err != nil {
						return def, fmt.Errorf("failed to convert item to entity input: %w", err)
					}

					doesItemNeedToBeChanged := !curr.Equal(newOnlyForComparisonWithInvalidEntitlement)

					if doesItemNeedToBeChanged {
						if currentPhaseIntact {
							// This means deleting the item with all its sub-resources
							if err := s.deleteItem(ctx, currentItemView); err != nil {
								return def, fmt.Errorf("failed to delete item: %w", err)
							}
						}

						currentItemIntact = false
					}

					var entitlementForNewItem *entitlement.Entitlement

					// Let's not pollute the scope
					{
						// Let's figure out what the cadence for the new entitlement should be
						cadenceForEntitlement := cadenceForItem

						hasCurrEnt := currentItemView.Entitlement != nil
						newEntInp, hasNewEnt, err := matchingItemFromNewSpec.ToScheduleSubscriptionEntitlementInput(
							view.Customer,
							cadenceForEntitlement,
						)
						if err != nil {
							return def, fmt.Errorf("failed to determine entitlement input for item %s: %w", currentItemView.SubscriptionItem.Key, err)
						}

						currentEntitlementIntact := currentItemIntact && hasCurrEnt // TODO: all "intact"s can only ever set to be false, create a custom type that ensures it

						// We need to delete the current entitlement if it exists, and the new would be nil or different
						if currentEntitlementIntact {
							if !hasNewEnt {
								if err := s.EntitlementAdapter.DeleteByItemID(ctx, currentItemView.SubscriptionItem.NamespacedID); err != nil {
									return def, fmt.Errorf("failed to delete entitlement: %w", err)
								}

								currentEntitlementIntact = false
							} else {
								// Let's compare if it needs changing
								// We can compare the two to see if it needs changing
								// We have to be careful of feature comparison, the current will have feature ID informatino while the new will not
								currToCompare := currentItemView.Entitlement.ToScheduleSubscriptionEntitlementInput()
								if err := newEntInp.CreateEntitlementInputs.Validate(); err != nil {
									return def, fmt.Errorf("failed to validate new entitlement input: %w", err)
								}

								if newEntInp.CreateEntitlementInputs.FeatureID == nil {
									currToCompare.CreateEntitlementInputs.FeatureID = nil
								} else if newEntInp.CreateEntitlementInputs.FeatureKey == nil {
									currToCompare.CreateEntitlementInputs.FeatureKey = nil
								}

								if !currToCompare.Equal(newEntInp) || doesItemNeedToBeChanged {
									// First we need to delete the old entitlement
									if err := s.EntitlementAdapter.DeleteByItemID(ctx, currentItemView.SubscriptionItem.NamespacedID); err != nil {
										return def, fmt.Errorf("failed to delete entitlement: %w", err)
									}

									currentEntitlementIntact = false
								}
							}
						}

						// We need to create the entitlement, if any previous has already been deleted and the new one exists
						if !currentEntitlementIntact && hasNewEnt {
							sEnt, err := s.EntitlementAdapter.ScheduleEntitlement(ctx, newEntInp)
							if err != nil {
								return def, fmt.Errorf("failed to create entitlement: %w", err)
							}

							entitlementForNewItem = &sEnt.Entitlement
						}
					}

					if !currentItemIntact {
						// Then we also need to recreate the item
						new, err := matchingItemFromNewSpec.ToCreateSubscriptionItemEntityInput(
							phaseToLinkToForNewResources,
							cadenceOfNewPhaseBasedOnSpec,
							entitlementForNewItem,
						)
						if err != nil {
							return def, fmt.Errorf("failed to convert item to entity input: %w", err)
						}

						if _, err := s.SubscriptionItemRepo.Create(ctx, new); err != nil {
							return def, fmt.Errorf("failed to create item: %w", err)
						}
					}
				}
			}
		}

		// 3. Now, we have to check for any new phases and items as those were left out by the previous logic
		for _, phase := range newSpec.GetSortedPhases() {
			// Sanity check
			if phase == nil {
				return def, fmt.Errorf("phase is nil")
			}

			// Let's see if the phase was present in the current view
			matchingPhaseInCurrentView, foundMatchingPhaseInCurrentView := lo.Find(view.Phases, func(p subscription.SubscriptionPhaseView) bool {
				return p.SubscriptionPhase.Key == phase.PhaseKey
			})

			if !foundMatchingPhaseInCurrentView {
				phaseCadence, err := newSpec.GetPhaseCadence(phase.PhaseKey)
				if err != nil {
					return def, fmt.Errorf("failed to get cadence for phase %s: %w", phase.PhaseKey, err)
				}

				if _, err := s.createPhase(ctx, view.Customer, *phase, view.Subscription, phaseCadence); err != nil {
					return def, fmt.Errorf("failed to create phase: %w", err)
				}
				continue
			}

			// Now lets check all the items in the phase
			for key, itemsByKey := range phase.ItemsByKey {
				matchingItemsByKeyInCurrentView, foundMatchingItemsByKeyInCurrentView := matchingPhaseInCurrentView.ItemsByKey[key]

				for itemIdx, item := range itemsByKey {
					phaseCadence, err := newSpec.GetPhaseCadence(phase.PhaseKey)
					if err != nil {
						return def, fmt.Errorf("failed to get cadence for phase %s: %w", phase.PhaseKey, err)
					}

					itemCadence, err := item.GetCadence(phaseCadence)
					if err != nil {
						return def, fmt.Errorf("failed to get cadence for item %s: %w", item.ItemKey, err)
					}

					// If we didn't find a matching key in the current view, we need to create the item
					if !foundMatchingItemsByKeyInCurrentView {
						if _, err := s.createItem(
							ctx,
							view.Customer,
							item,
							matchingPhaseInCurrentView.SubscriptionPhase,
							itemCadence,
						); err != nil {
							return def, fmt.Errorf("failed to create item: %w", err)
						}

						// There's nothing left to do for this item
						continue
					} else if itemIdx >= len(matchingItemsByKeyInCurrentView) {
						// If there's a matching key, then in the previous step we've taken care of all indexes
						// present in the current phase

						// The rest we create
						if _, err := s.createItem(
							ctx,
							view.Customer,
							item,
							matchingPhaseInCurrentView.SubscriptionPhase,
							itemCadence,
						); err != nil {
							return def, fmt.Errorf("failed to create item: %w", err)
						}
					}
				}
			}
		}

		// 4. Finally we're done with syncing everything, we should just re-fetch the subscription
		return s.Get(ctx, view.Subscription.NamespacedID)
	})
}
