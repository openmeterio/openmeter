package price

import (
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/pkg/models"
)

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

	Value plan.Price `json:"value,omitempty"`
}

type CreateInput struct {
	Spec
	models.CadencedModel
	SubscriptionId models.NamespacedID `json:"subscriptionId,omitempty"`
}

type Spec struct {
	PhaseKey string     `json:"phaseKey,omitempty"`
	ItemKey  string     `json:"itemKey,omitempty"`
	Value    plan.Price `json:"value,omitempty"`
	Key      string     `json:"key,omitempty"`
}

type NotFoundError struct {
	ID string
}

func (e NotFoundError) Error() string {
	return "price with id " + e.ID + " not found"
}
