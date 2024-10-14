package price

import "github.com/openmeterio/openmeter/pkg/models"

type Price struct {
	models.NamespacedModel
	models.ManagedModel
	models.CadencedModel

	ID  string `json:"id,omitempty"`
	Key string `json:"key,omitempty"`

	// References to find the price by
	SubscriptionId string `json:"subscriptionId,omitempty"`
	PhaseKey       string `json:"phaseKey,omitempty"`
	ItemKey        string `json:"itemKey,omitempty"`

	// String representation of the numerical fix point decimal value
	Value string `json:"value,omitempty"`
}

type CreateInput struct {
	Spec
	SubscriptionId models.NamespacedID `json:"subscriptionId,omitempty"`
}

type Spec struct {
	PhaseKey string `json:"phaseKey,omitempty"`
	ItemKey  string `json:"itemKey,omitempty"`
	Value    string `json:"value,omitempty"`
}
