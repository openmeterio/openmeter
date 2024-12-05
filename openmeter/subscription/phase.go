package subscription

import (
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

type SubscriptionPhase struct {
	models.NamespacedID
	models.ManagedModel

	ActiveFrom time.Time `json:"activeFrom"`

	// SubscriptionID is the ID of the subscription this phase belongs to.
	SubscriptionID string `json:"subscriptionId"`

	// Key is the unique key for Phase.
	Key string `json:"key"`

	// Name
	Name string `json:"name"`

	// Description
	Description *string `json:"description,omitempty"`

	// Metadata
	Metadata map[string]string `json:"metadata,omitempty"`
}
