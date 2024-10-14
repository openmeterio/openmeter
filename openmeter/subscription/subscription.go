package subscription

import (
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

type PlanRef struct {
	Key     string
	Version int
}

type Subscription struct {
	models.NamespacedModel
	models.ManagedModel
	models.CadencedModel

	ID string `json:"id,omitempty"`
	SubscriptionSpec
}

type SubscriptionPatch struct {
	models.NamespacedModel
	models.ManagedModel

	ID             string `json:"id,omitempty"`
	SubscriptionId string `json:"subscriptionId,omitempty"`

	// Primary ordering happens via activation time
	ActiveFrom time.Time `json:"activeFrom,omitempty"`
	// Secondary ordering can be used as a tie-breaker
	SecondaryOrdering int `json:"secondaryOrdering,omitempty"`

	// Patch info
	Operation string `json:"operation,omitempty"`
	Path      string `json:"path,omitempty"`
	Value     string `json:"value,omitempty"`
}
