package subscription

import (
	"github.com/openmeterio/openmeter/pkg/models"
)

// TODO[galexi]: Implement this
func ValidateUniqueConstraintByFeatures(subs []SubscriptionSpec) error {
	return nil
}

func ValidateUniqueConstraintBySubscriptions(subs []SubscriptionSpec) error {
	if overlaps := models.NewSortedCadenceList(subs).GetOverlaps(); len(overlaps) > 0 {
		return ErrOnlySingleSubscriptionAllowed.WithAttrs(models.Attributes{
			"overlaps": overlaps,
		})
	}

	return nil
}
