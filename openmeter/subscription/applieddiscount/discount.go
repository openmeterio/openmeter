package applieddiscount

import "github.com/openmeterio/openmeter/pkg/models"

type AppliedDiscount struct {
	models.NamespacedModel
	models.ManagedModel

	ID string `json:"id,omitempty"`

	SubscriptionId string `json:"subscriptionId,omitempty"`
	PhaseKey       string `json:"phaseKey,omitempty"`

	Amount any
	// List of SubscriptionItemKeys used as a filter when applying the discount.
	// If none is present the discount applies to everything in the Phase.
	AppliesTo []string `json:"appliesTo,omitempty"`
}

type CreateInput struct {
	Spec
	SubscriptionId models.NamespacedID `json:"subscriptionId,omitempty"`
}

type Spec struct {
	PhaseKey  string   `json:"phaseKey,omitempty"`
	AppliesTo []string `json:"appliesTo,omitempty"`
}
