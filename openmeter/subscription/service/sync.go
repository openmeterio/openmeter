package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Sync manages the synchronization of a subscription with a new spec.
// It consists of 3 steps:
// 1. Remove anything that's changed or got removed
// 2. Create anything that's been changed
// 3. Create anything that's new
//
// Some remarks:
// 1. Change comparison is done on the relevant create inputs.
// 2. Things being deleted are marked via a `touched` map.
//
// TODO: localize error so phase and item keys are always included (alongside subscription reference)
// TODO (OM-1074): clean up this control flow
func (s *service) sync(ctx context.Context, view subscription.SubscriptionView, newSpec subscription.SubscriptionSpec) (subscription.Subscription, error) {
	setSpanAttrs(ctx,
		attribute.String("subscription.namespace", view.Subscription.Namespace),
		attribute.String("subscription.id", view.Subscription.ID),
		attribute.String("subscription.sync.operation", "spec_sync"),
	)
	setSpanAttrs(ctx, addViewAttrs([]attribute.KeyValue{}, "subscription.view.before", view)...)
	setSpanAttrs(ctx, addSpecAttrs([]attribute.KeyValue{}, "subscription.spec.before", view.Spec)...)
	setSpanAttrs(ctx, addSpecAttrs([]attribute.KeyValue{}, "subscription.spec.target", newSpec)...)

	return transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (subscription.Subscription, error) {
		var def subscription.Subscription
		var phaseDeleted, phaseCreated int
		var itemDeleted, itemCreated int

		// Some sanity checks for good measure
		if view.Subscription.CustomerId != newSpec.CustomerId {
			return def, fmt.Errorf("cannot change customer id")
		}
		if !view.Subscription.PlanRef.NilEqual(newSpec.Plan) {
			return def, fmt.Errorf("cannot change plan")
		}
		if !view.Subscription.ActiveFrom.Equal(newSpec.ActiveFrom) {
			return def, fmt.Errorf("cannot change subscription start")
		}

		dirty := make(touched)

		// Let's make sure the Subscription Cadence is up to date
		if !view.Subscription.CadencedModel.Equal(models.CadencedModel{ActiveFrom: newSpec.ActiveFrom, ActiveTo: newSpec.ActiveTo}) {
			_, err := s.SubscriptionRepo.SetEndOfCadence(ctx, view.Subscription.NamespacedID, newSpec.ActiveTo)
			if err != nil {
				return def, fmt.Errorf("failed to set end of cadence: %w", err)
			}
		}

		// 1. Let's remove anything that's changed or got removed
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
				phaseDeleted++
				addSpanEvent(ctx, "subscription.sync.phase.delete",
					attribute.String("phase.key", currentPhaseView.SubscriptionPhase.Key),
					attribute.String("phase.id", currentPhaseView.SubscriptionPhase.ID),
					attribute.String("reason", "removed"),
				)

				dirty.mark(subscription.NewPhasePath(currentPhaseView.SubscriptionPhase.Key))

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

			// If the phase has any changes, we need to recreate it. That means also all sub-resources of it have to be relinked.
			if !curr.Equal(new) {
				// This means deleting the phase with all its sub-resources
				if err := s.deletePhase(ctx, currentPhaseView); err != nil {
					return def, fmt.Errorf("failed to delete phase: %w", err)
				}
				phaseDeleted++
				addSpanEvent(ctx, "subscription.sync.phase.delete",
					attribute.String("phase.key", currentPhaseView.SubscriptionPhase.Key),
					attribute.String("phase.id", currentPhaseView.SubscriptionPhase.ID),
					attribute.String("reason", "changed"),
				)

				dirty.mark(subscription.NewPhasePath(currentPhaseView.SubscriptionPhase.Key))

				// The phase is deleted, there's nothing more to be done
				continue
			}

			// Sanity check, the current phase cannot be dirty
			if dirty.isTouched(subscription.NewPhasePath(currentPhaseView.SubscriptionPhase.Key)) {
				return def, fmt.Errorf("current phase is dirty but should not be")
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
						if err := s.deleteItem(ctx, currentItemView); err != nil {
							return def, fmt.Errorf("failed to delete item: %w", err)
						}
						itemDeleted++
						addSpanEvent(ctx, "subscription.sync.item.delete",
							attribute.String("phase.key", currentItemView.Spec.PhaseKey),
							attribute.String("item.key", currentItemView.Spec.ItemKey),
							attribute.String("item.id", currentItemView.SubscriptionItem.ID),
							attribute.String("reason", "key_removed"),
						)

						dirty.mark(subscription.NewItemPath(currentItemView.Spec.PhaseKey, currentItemView.Spec.ItemKey))
					}

					// There's nothing more to be done for this item(key), so lets skip to the next one
					continue
				}

				for currentItemIdx, currentItemView := range currentItemViews {
					// Let's get the item with the same index from the new spec
					if currentItemIdx >= len(matchingItemsByKeyFromNewSpec) {
						// Let's delete the item as it's not present in the new spec

						if err := s.deleteItem(ctx, currentItemView); err != nil {
							return def, fmt.Errorf("failed to delete item: %w", err)
						}
						itemDeleted++
						addSpanEvent(ctx, "subscription.sync.item.delete",
							attribute.String("phase.key", currentItemView.Spec.PhaseKey),
							attribute.String("item.key", currentItemView.Spec.ItemKey),
							attribute.String("item.id", currentItemView.SubscriptionItem.ID),
							attribute.Int("item.version", currentItemIdx),
							attribute.String("reason", "version_removed"),
						)

						dirty.mark(subscription.NewItemVersionPath(currentItemView.Spec.PhaseKey, currentItemView.Spec.ItemKey, currentItemIdx))

						// There's nothing more to be done for this item, so lets skip to the next one
						continue
					}

					matchingItemFromNewSpec := matchingItemsByKeyFromNewSpec[currentItemIdx]

					// First, let's check if the item itself needs to be changed
					curr, err := currentItemView.Spec.ToCreateSubscriptionItemEntityInput(
						currentPhaseView.SubscriptionPhase.NamespacedID,
						cadenceOfCurrentPhaseBasedOnSpec,
						convert.SafeDeRef(currentItemView.Entitlement, func(s subscription.SubscriptionEntitlement) *entitlement.Entitlement {
							return &s.Entitlement.Entitlement
						}),
					)
					if err != nil {
						return def, fmt.Errorf("failed to convert item to entity input: %w", err)
					}

					// Here we don't preamptively know all the properties but fortunately all we need to know is whether they'd change or not
					// We're prepopulating changing fields with invalid values, which is a lie and a bad method, but it's necessary for now due to the hard linking

					newPhaseID := currentPhaseView.SubscriptionPhase.NamespacedID
					if dirty.isTouched(subscription.NewPhasePath(currentPhaseView.SubscriptionPhase.Key)) {
						newPhaseID = impossibleNamespacedId
					}

					newOnlyForComparisonWithInvalidProperties, err := matchingItemFromNewSpec.ToCreateSubscriptionItemEntityInput(
						newPhaseID,
						cadenceOfNewPhaseBasedOnSpec,
						// Without the "new" entitlement already present we cannot properly compare the two create inputs.
						// To work around this, we'll reuse the current entitlement for comparison.
						// This won't cause an issue as all relevant properties of the entitlement are specced on the Item (as part of RateCard)
						// FIXME: This is a lie
						convert.SafeDeRef(currentItemView.Entitlement, func(s subscription.SubscriptionEntitlement) *entitlement.Entitlement {
							return &s.Entitlement.Entitlement
						}),
					)
					if err != nil {
						return def, fmt.Errorf("failed to convert item to entity input: %w", err)
					}

					doesItemNeedToBeChanged := !curr.Equal(newOnlyForComparisonWithInvalidProperties)

					if doesItemNeedToBeChanged {
						// This means deleting the item with all its sub-resources
						if err := s.deleteItem(ctx, currentItemView); err != nil {
							return def, fmt.Errorf("failed to delete item: %w", err)
						}
						itemDeleted++
						addSpanEvent(ctx, "subscription.sync.item.delete",
							attribute.String("phase.key", currentItemView.Spec.PhaseKey),
							attribute.String("item.key", currentItemView.Spec.ItemKey),
							attribute.String("item.id", currentItemView.SubscriptionItem.ID),
							attribute.Int("item.version", currentItemIdx),
							attribute.String("reason", "changed"),
						)

						dirty.mark(subscription.NewItemVersionPath(currentItemView.Spec.PhaseKey, currentItemView.Spec.ItemKey, currentItemIdx))

						// There's nothing more to be done here, so lets skip to the next one
						continue
					}
				}
			}
		}

		// 2. Let's create anything that's been changed
		for _, currentPhaseView := range view.Phases {
			// Let's try find a matching phase in the new spec
			matchingPhaseFromNewSpec, found := lo.Find(newSortedPhaseSpecs, func(s *subscription.SubscriptionPhaseSpec) bool {
				return s.PhaseKey == currentPhaseView.SubscriptionPhase.Key
			})

			if !found {
				// If the phase wasn't found there's nothing to create
				continue
			}

			// Sanity check
			if matchingPhaseFromNewSpec == nil {
				return def, fmt.Errorf("failed to find matching phase in new spec but no error was returned")
			}

			newPhaseCadence, err := newSpec.GetPhaseCadence(matchingPhaseFromNewSpec.PhaseKey)
			if err != nil {
				return def, fmt.Errorf("failed to get cadence for phase %s: %w", matchingPhaseFromNewSpec.PhaseKey, err)
			}

			// If the phase got deleted, we can create it as a whole
			if dirty.isTouched(subscription.NewPhasePath(currentPhaseView.SubscriptionPhase.Key)) {
				if _, err := s.createPhase(ctx, view.Customer, *matchingPhaseFromNewSpec, view.Subscription, newPhaseCadence); err != nil {
					return def, fmt.Errorf("failed to create phase: %w", err)
				}
				phaseCreated++

				// There's nothing more to be done for this phase, so lets skip to the next one
				continue
			}

			// Now let's check each of the items in the phase
			for currentItemViewsKey, currentItemViews := range currentPhaseView.ItemsByKey {
				// Let's try find a matching item in the new spec
				// Here as we do an update, we rely on the previously verified integrity of both view and spec
				// Due to this, we use a simple matching based on the index of the item under the given key
				matchingItemsByKeyFromNewSpec, found := matchingPhaseFromNewSpec.ItemsByKey[currentItemViewsKey]
				if !found {
					// If the item wasn't found there's nothing to create
					continue
				}

				for currentItemIdx, currentItemView := range currentItemViews {
					// Let's get the item with the same index from the new spec
					if currentItemIdx >= len(matchingItemsByKeyFromNewSpec) {
						// We went out of bounds, these items are not present in the new spec so we can just break

						break
					}

					matchingItemFromNewSpec := matchingItemsByKeyFromNewSpec[currentItemIdx]

					// If the item got deleted, we can create it as a whole
					if dirty.isTouched(subscription.NewItemVersionPath(currentItemView.Spec.PhaseKey, currentItemView.Spec.ItemKey, currentItemIdx)) {
						if _, err := s.createItem(ctx, createItemOptions{
							cust:         view.Customer,
							sub:          view.Subscription,
							phase:        currentPhaseView.SubscriptionPhase,
							phaseCadence: newPhaseCadence,
							itemSpec:     *matchingItemFromNewSpec,
						}); err != nil {
							return def, fmt.Errorf("failed to create item: %w", err)
						}
						itemCreated++

						// There's nothing more to be done for this item, so lets skip to the next one
						continue
					}
				}
			}
		}

		// 3. Finally, let's create anything that's new
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
				phaseCreated++
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

					// If we didn't find a matching key in the current view, we need to create the item
					if !foundMatchingItemsByKeyInCurrentView {
						if _, err := s.createItem(ctx, createItemOptions{
							cust:         view.Customer,
							sub:          view.Subscription,
							phase:        matchingPhaseInCurrentView.SubscriptionPhase,
							phaseCadence: phaseCadence,
							itemSpec:     *item,
						}); err != nil {
							return def, fmt.Errorf("failed to create item: %w", err)
						}
						itemCreated++

						// There's nothing left to do for this item
						continue
					} else if itemIdx >= len(matchingItemsByKeyInCurrentView) {
						// If there's a matching key, then in the previous step we've taken care of all indexes
						// present in the current phase

						// The rest we create
						if _, err := s.createItem(ctx, createItemOptions{
							cust:         view.Customer,
							sub:          view.Subscription,
							phase:        matchingPhaseInCurrentView.SubscriptionPhase,
							phaseCadence: phaseCadence,
							itemSpec:     *item,
						}); err != nil {
							return def, fmt.Errorf("failed to create item: %w", err)
						}
						itemCreated++
					}
				}
			}
		}

		// 4. Finally we're done with syncing everything, we should just re-fetch the subscription
		setSpanAttrs(ctx,
			attribute.Int("subscription.sync.touched_paths.count", len(dirty)),
			attribute.Int("subscription.sync.phases.deleted", phaseDeleted),
			attribute.Int("subscription.sync.phases.created", phaseCreated),
			attribute.Int("subscription.sync.items.deleted", itemDeleted),
			attribute.Int("subscription.sync.items.created", itemCreated),
		)

		sub, err := s.Get(ctx, view.Subscription.NamespacedID)
		if err != nil {
			return def, err
		}
		setSpanAttrs(ctx,
			attribute.String("subscription.sync.result_id", sub.ID),
			attribute.String("subscription.sync.result_namespace", sub.Namespace),
		)

		return sub, nil
	})
}

// touched is a map of touched paths (honoring sub-resource relationships)
type touched map[subscription.SpecPath]bool

// Mark a given path as touched
func (t touched) mark(key subscription.SpecPath) {
	t[key] = true
}

// Check if a given path has been touched
// If path X has been touched, then all sub-resources of X have been touched
func (t touched) isTouched(key subscription.SpecPath) bool {
	for k := range t {
		// IsParentOf check returns true for identity as well
		if k.IsParentOf(key) {
			return true
		}
	}
	return false
}

// NewItemVersionPath returns an invalid PatchPath thats still usable for IsParentOf checks
// FIXME: this is a hack. For instance, is featureKey were to contain `/` it would completely break (though that exact scenario is otherwise prohibited)

// an ID that can never occur in normal control flow
var impossibleNamespacedId = models.NamespacedID{
	ID:        "impossible",
	Namespace: "impossible",
}
