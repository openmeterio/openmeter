package subscriptionaddon

import (
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type SubscriptionAddonQuantity struct {
	models.NamespacedID
	models.ManagedModel

	ActiveFrom time.Time `json:"activeFrom"`
	Quantity   int       `json:"quantity"`
}

func (q SubscriptionAddonQuantity) AsTimed() timeutil.Timed[SubscriptionAddonQuantity] {
	return timeutil.AsTimed(func(q SubscriptionAddonQuantity) time.Time {
		return q.ActiveFrom
	})(q)
}

type CreateSubscriptionAddonQuantityInput struct {
	ActiveFrom time.Time `json:"activeFrom"`
	Quantity   int       `json:"quantity"`
}
