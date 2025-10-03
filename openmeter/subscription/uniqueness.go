package subscription

import (
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

// TODO[galexi]: Implement this
func ValidateUniqueConstraintByFeatures(subs []SubscriptionSpec) error {
	return nil
}

func ValidateUniqueConstraintBySubscriptions(subs []SubscriptionSpec) error {
	if overlaps := models.NewSortedCadenceList(
		slicesx.Map(subs, func(i SubscriptionSpec) CreateSubscriptionEntityInput {
			return i.ToCreateSubscriptionEntityInput("irrelevant")
		}),
	).GetOverlaps(); len(overlaps) > 0 {
		return ErrOnlySingleSubscriptionAllowed
	}

	return nil
}
