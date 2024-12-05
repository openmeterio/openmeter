package subscription

import (
	"fmt"

	"github.com/openmeterio/openmeter/pkg/datex"
	"github.com/openmeterio/openmeter/pkg/models"
)

type SubscriptionItem struct {
	models.NamespacedID
	models.ManagedModel
	models.AnnotatedModel

	// SubscriptionItem doesn't have a separate Cadence, only one relative to the phase, denoting if it's intentionally different from the phase's cadence.
	// The durations are relative to phase start.
	ActiveFromOverrideRelativeToPhaseStart *datex.Period `json:"activeFromOverrideRelativeToPhaseStart,omitempty"`
	ActiveToOverrideRelativeToPhaseStart   *datex.Period `json:"activeToOverrideRelativeToPhaseStart,omitempty"`

	// The defacto cadence of the item is calculated and persisted after each change.
	models.CadencedModel

	// SubscriptionID is the ID of the subscription this item belongs to.
	SubscriptionId string `json:"subscriptionId"`
	// PhaseID is the ID of the phase this item belongs to.
	PhaseId string `json:"phaseId"`
	// Key is the unique key of the item in the phase.
	Key string `json:"itemKey"`

	RateCard RateCard `json:"rateCard"`

	EntitlementID *string `json:"entitlementId,omitempty"`
}

func (i SubscriptionItem) GetCadence(phaseCadence models.CadencedModel) models.CadencedModel {
	start := phaseCadence.ActiveFrom

	if i.ActiveFromOverrideRelativeToPhaseStart != nil {
		start, _ = i.ActiveFromOverrideRelativeToPhaseStart.AddTo(start)
	}

	end := phaseCadence.ActiveTo

	if i.ActiveToOverrideRelativeToPhaseStart != nil {
		iEnd, _ := i.ActiveToOverrideRelativeToPhaseStart.AddTo(start)

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
		ActiveFromOverrideRelativeToPhaseStart: i.ActiveFromOverrideRelativeToPhaseStart,
		ActiveToOverrideRelativeToPhaseStart:   i.ActiveToOverrideRelativeToPhaseStart,
		PhaseID:                                i.PhaseId,
		Key:                                    i.Key,
		RateCard:                               i.RateCard,
		EntitlementID:                          i.EntitlementID,
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

// SubscriptionItems under the same key in a phase have to meet this criteria
func ValidateCadencesAreSortedAndNonOverlapping(cadences []models.CadencedModel) error {
	// First, let's validate that the cadences are sorted by ActiveFrom
	for i := range cadences {
		if i == 0 {
			continue
		}

		if cadences[i-1].ActiveFrom.After(cadences[i].ActiveFrom) {
			return fmt.Errorf("cadences at %d and %d are not sorted by ActiveFrom: %s > %s", i-1, i, cadences[i-1].ActiveFrom, cadences[i].ActiveFrom)
		}

		if cadences[i-1].ActiveTo == nil {
			return fmt.Errorf("cadence %d overlaps cadence %d starting %s due to missing ActiveTo", i-1, i, cadences[i-1].ActiveFrom)
		}

		if cadences[i-1].ActiveTo.After(cadences[i].ActiveFrom) {
			return fmt.Errorf("cadence %d overlaps cadence %d between %s and %s", i-1, i, cadences[i].ActiveFrom, cadences[i-1].ActiveTo)
		}
	}

	return nil
}
