package subscription

import (
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/models"
	modelref "github.com/openmeterio/openmeter/pkg/models/ref"
)

type ContentOverride struct {
	ContentCreateInput
}

type Content struct {
	models.NamespacedModel
	models.ManagedModel
	models.CadencedModel

	ID             string         `json:"id"`
	SubscriptionID modelref.IDRef `json:"subscriptionId"`

	// References the PlanContent that was used to template this Content.
	// If empty, the content is not templated.
	PlanContentRef *modelref.IDRef `json:"planContentRef,omitempty"`
}

type ContentCreateInput struct {
	models.NamespacedModel
	models.CadencedModel

	// References the PlanContent that was used to template this Content.
	// If empty, the content is not templated.
	PlanContentRef *modelref.IDRef `json:"planContentRef,omitempty"`
}

func ContentToEntitlementCreateInput(c Content) entitlement.CreateEntitlementInputs {
	panic("implement me")
}
