package subscription

import (
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/models"
)

// ValidateSubscriptionTimeline checks if the given subscriptions form a valid timeline without overlaps
func ValidateSubscriptionTimeline(subscriptions []Subscription) error {
	// Convert subscriptions to entity inputs for the cadence list
	subscriptionInputs := lo.Map(subscriptions, func(sub Subscription, _ int) CreateSubscriptionEntityInput {
		return sub.AsEntityInput()
	})

	// Create a sorted cadence list
	subscriptionTimeline := models.NewSortedCadenceList(subscriptionInputs)

	// Check for overlaps
	if overlaps := subscriptionTimeline.GetOverlaps(); len(overlaps) > 0 {
		return fmt.Errorf("subscription timeline has overlaps: %v", overlaps)
	}

	return nil
}

// ValidateSubscriptionTimelineWithNew checks if adding a new subscription to the existing ones
// would result in a valid timeline without overlaps
func ValidateSubscriptionTimelineWithNew(existingSubscriptions []Subscription, newSpec SubscriptionSpec, namespace string) error {
	// Convert existing subscriptions to entity inputs
	subscriptionInputs := lo.Map(existingSubscriptions, func(sub Subscription, _ int) CreateSubscriptionEntityInput {
		return sub.AsEntityInput()
	})

	// Add the new subscription spec
	subscriptionInputs = append(subscriptionInputs, newSpec.ToCreateSubscriptionEntityInput(namespace))

	// Create a sorted cadence list
	subscriptionTimeline := models.NewSortedCadenceList(subscriptionInputs)

	// Check for overlaps
	if overlaps := subscriptionTimeline.GetOverlaps(); len(overlaps) > 0 {
		return models.NewGenericConflictError(fmt.Errorf("new subscription would overlap with existing ones: %v", overlaps))
	}

	return nil
}
