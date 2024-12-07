package subscription

import (
	"time"

	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Subscription struct {
	models.NamespacedID
	models.ManagedModel
	models.CadencedModel

	Plan PlanRef

	CustomerId string `json:"customerId,omitempty"`
	Currency   currencyx.Code
}

func (s Subscription) GetStatusAt(at time.Time) SubscriptionStatus {
	// Cadence might not be initialized
	if s.CadencedModel.IsZero() {
		return SubscriptionStatusInactive
	}

	// If the subscription has already started...
	if !s.ActiveFrom.After(at) {
		// ...and it has not been canceled yet, it is active
		if s.ActiveTo == nil {
			return SubscriptionStatusActive
		}
		// ...and it has been canceled, it is canceled
		if s.ActiveTo.After(at) {
			return SubscriptionStatusCanceled
		}
	}

	// The default status is inactive
	return SubscriptionStatusInactive
}
