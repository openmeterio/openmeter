package subscription

import (
	"time"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Subscription struct {
	models.NamespacedID
	models.ManagedModel
	models.CadencedModel
	models.MetadataModel

	productcatalog.Alignment

	Name        string  `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`

	// References the plan (if the Subscription was created form one)
	PlanRef *PlanRef `json:"planRef"`

	CustomerId string         `json:"customerId,omitempty"`
	Currency   currencyx.Code `json:"currency,omitempty"`

	BillingCadence  isodate.Period                 `json:"billing_cadence"`
	BillingAnchor   time.Time                      `json:"billingAnchor"`
	ProRatingConfig productcatalog.ProRatingConfig `json:"pro_rating_config"`
}

func (s Subscription) AsEntityInput() CreateSubscriptionEntityInput {
	return CreateSubscriptionEntityInput{
		CadencedModel: s.CadencedModel,
		NamespacedModel: models.NamespacedModel{
			Namespace: s.Namespace,
		},
		Alignment:       s.Alignment,
		MetadataModel:   s.MetadataModel,
		Plan:            s.PlanRef,
		Name:            s.Name,
		Description:     s.Description,
		CustomerId:      s.CustomerId,
		Currency:        s.Currency,
		BillingCadence:  s.BillingCadence,
		ProRatingConfig: s.ProRatingConfig,
	}
}

func (s Subscription) GetStatusAt(at time.Time) SubscriptionStatus {
	// Cadence might not be initialized
	if s.CadencedModel.IsZero() {
		return SubscriptionStatusInactive
	}

	if s.DeletedAt != nil && !s.DeletedAt.After(at) {
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
	} else {
		// If the subscription is scheduled to start in the future, it is scheduled
		return SubscriptionStatusScheduled
	}

	// The default status is inactive
	return SubscriptionStatusInactive
}

func (s Subscription) GetCustomerID() customer.CustomerID {
	return customer.CustomerID{
		Namespace: s.Namespace,
		ID:        s.CustomerId,
	}
}
