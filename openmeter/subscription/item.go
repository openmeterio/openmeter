package subscription

import (
	"github.com/openmeterio/openmeter/pkg/models"
)

type SubscriptionItem struct {
	models.NamespacedID   `json:",inline"`
	models.ManagedModel   `json:",inline"`
	models.AnnotatedModel `json:",inline"`

	// SubscriptionItem doesn't have a separate Cadence, only one relative to the phase, denoting if it's intentionally different from the phase's cadence.
	CadenceOverrideRelativeToPhaseStart

	// The defacto cadence of the item is calculated and persisted after each change.
	models.CadencedModel `json:",inline"`

	// SubscriptionID is the ID of the subscription this item belongs to.
	SubscriptionId string `json:"subscriptionId"`
	// PhaseID is the ID of the phase this item belongs to.
	PhaseId string `json:"phaseId"`
	// Key is the unique key of the item in the phase.
	Key string `json:"itemKey"`

	RateCard RateCard `json:"rateCard"`

	EntitlementID *string `json:"entitlementId,omitempty"`
	// Name
	Name string `json:"name"`

	// Description
	Description *string `json:"description,omitempty"`
}

func (i SubscriptionItem) GetCadence(phaseCadence models.CadencedModel) models.CadencedModel {
	start := phaseCadence.ActiveFrom

	if i.CadenceOverrideRelativeToPhaseStart.ActiveFromOverride != nil {
		start, _ = i.CadenceOverrideRelativeToPhaseStart.ActiveFromOverride.AddTo(start)
	}

	end := phaseCadence.ActiveTo

	if i.CadenceOverrideRelativeToPhaseStart.ActiveToOverride != nil {
		iEnd, _ := i.CadenceOverrideRelativeToPhaseStart.ActiveToOverride.AddTo(start)

		if end == nil || iEnd.Before(*end) {
			end = &iEnd
		}
	}

	return models.CadencedModel{
		ActiveFrom: start,
		ActiveTo:   end,
	}
}

func (i SubscriptionItem) AsEntityInput() CreateSubscriptionItemEntityInput {
	return CreateSubscriptionItemEntityInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: i.Namespace,
		},
		AnnotatedModel:                         i.AnnotatedModel,
		CadencedModel:                          i.CadencedModel,
		ActiveFromOverrideRelativeToPhaseStart: i.CadenceOverrideRelativeToPhaseStart.ActiveFromOverride,
		ActiveToOverrideRelativeToPhaseStart:   i.CadenceOverrideRelativeToPhaseStart.ActiveToOverride,
		PhaseID:                                i.PhaseId,
		Key:                                    i.Key,
		RateCard:                               i.RateCard,
		EntitlementID:                          i.EntitlementID,
		Name:                                   i.Name,
		Description:                            i.Description,
	}
}

// SubscriptionItemRef is an unstable reference to a SubscriptionItem
type SubscriptionItemRef struct {
	SubscriptionId string `json:"subscriptionId"`
	PhaseKey       string `json:"phaseKey"`
	ItemKey        string `json:"itemKey"`
}

func (r SubscriptionItemRef) Equals(r2 SubscriptionItemRef) bool {
	if r.SubscriptionId != r2.SubscriptionId {
		return false
	}
	if r.PhaseKey != r2.PhaseKey {
		return false
	}
	if r.ItemKey != r2.ItemKey {
		return false
	}
	return true
}
