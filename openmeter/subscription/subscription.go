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
	models.AnnotatedModel

	Name        string  `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`

	// References the plan (if the Subscription was created form one)
	PlanRef *PlanRef `json:"planRef"`

	CustomerId string         `json:"customerId,omitempty"`
	Currency   currencyx.Code `json:"currency,omitempty"`

	Verifications Verifications `json:"verifications"`
}

type Verifications struct {
	Payment Verification `json:"payment"`
}

func (v Verifications) IsVerified() bool {
	return v.Payment.IsVerified()
}

type Verification struct {
	Needed   bool `json:"needed"`
	Received bool `json:"received"`
}

func (v Verification) IsVerified() bool {
	return !v.Needed || v.Received
}

func (s Subscription) AsEntityInput() CreateSubscriptionEntityInput {
	return CreateSubscriptionEntityInput{
		CadencedModel: s.CadencedModel,
		NamespacedModel: models.NamespacedModel{
			Namespace: s.Namespace,
		},
		AnnotatedModel: s.AnnotatedModel,
		Plan:           s.PlanRef,
		Name:           s.Name,
		Description:    s.Description,
		CustomerId:     s.CustomerId,
		Currency:       s.Currency,
	}
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
			// ...if everything is verified, it is active
			if s.Verifications.IsVerified() {
				return SubscriptionStatusActive
			}

			// ...if verification is needed, it is pending
			return SubscriptionStatusPending
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
