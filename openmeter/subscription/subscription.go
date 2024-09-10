package subscription

import (
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	modelref "github.com/openmeterio/openmeter/pkg/models/ref"
)

type Subscription struct {
	models.NamespacedModel
	models.ManagedModel
	models.CadencedModel

	ID         string `json:"id,omitempty"`
	CustomerId string `json:"customerId,omitempty"`

	// The Key and Version of the Plan that was chosen when the Subscription was created.
	TemplatingPlanRef modelref.VersionedKeyRef `json:"templatingPlanRef,omitempty"`
}

func (s Subscription) IsActive() bool {
	return s.CadencedModel.IsActiveAt(clock.Now())
}
