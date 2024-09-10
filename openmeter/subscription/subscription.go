package subscription

import (
	"github.com/openmeterio/openmeter/pkg/models"
	modelref "github.com/openmeterio/openmeter/pkg/models/ref"
)

type Subscription struct {
	models.NamespacedModel
	models.ManagedModel
	models.CadencedModel
	models.VersionedModel

	ID string `json:"id,omitempty"`

	// The Key and Version of the Plan that was chosen when the Subscription was created.
	TemplatingPlanRef modelref.VersionedKeyRef `json:"templatingPlanRef,omitempty"`
	TrialConfig       TrialConfig
}

func (s Subscription) IsActive() bool {
	panic("implement me")
}

type SubscriptionCreateInput struct {
	models.NamespacedModel
	models.CadencedModel
	models.VersionedModel

	TemplatingPlanRef modelref.VersionedKeyRef
	TrialConfig       TrialConfig
}
type SubscriptionOverrides struct {
	// dummy
}

type TrialConfig struct{}

func (t TrialConfig) IsTrial() bool {
	panic("implement me")
}
