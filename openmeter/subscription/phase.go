package subscription

import (
	"time"

	"github.com/openmeterio/openmeter/pkg/datex"
	"github.com/openmeterio/openmeter/pkg/models"
)

type SubscriptionPhase struct {
	models.NamespacedID   `json:",inline"`
	models.ManagedModel   `json:",inline"`
	models.AnnotatedModel `json:",inline"`

	ActiveFrom time.Time `json:"activeFrom"`

	// SubscriptionID is the ID of the subscription this phase belongs to.
	SubscriptionID string `json:"subscriptionId"`

	// Key is the unique key for Phase.
	Key string `json:"key"`

	// Name
	Name string `json:"name"`

	// Description
	Description *string `json:"description,omitempty"`
}

type CadenceOverrideRelativeToPhaseStart struct {
	ActiveFromOverride *datex.Period `json:"activeFromOverrideRelativeToPhaseStart"`
	ActiveToOverride   *datex.Period `json:"activeToOverrideRelativeToPhaseStart,omitempty"`
}
