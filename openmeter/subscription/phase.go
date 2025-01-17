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

	// OrderedDiscountIDs is the list of discount IDs that are applied in order.
	OrderedDiscountIDs []string `json:"orderedDiscountIds"`
}

type CadenceOverrideRelativeToPhaseStart struct {
	ActiveFromOverride *datex.Period `json:"activeFromOverrideRelativeToPhaseStart"`
	ActiveToOverride   *datex.Period `json:"activeToOverrideRelativeToPhaseStart,omitempty"`
}

func (c CadenceOverrideRelativeToPhaseStart) GetCadence(base models.CadencedModel) models.CadencedModel {
	start := base.ActiveFrom
	if c.ActiveFromOverride != nil {
		start, _ = c.ActiveFromOverride.AddTo(base.ActiveFrom)
	}

	if base.ActiveTo != nil {
		if base.ActiveTo.Before(start) {
			// If the intended start time is after the intended end time of the phase, the item will have 0 lifetime at the end of the phase
			// This scenario is possible when Subscriptions are canceled (before the phase ends)
			return models.CadencedModel{
				ActiveFrom: *base.ActiveTo,
				ActiveTo:   base.ActiveTo,
			}
		}
	}

	end := base.ActiveTo

	if c.ActiveToOverride != nil {
		endTime, _ := c.ActiveToOverride.AddTo(base.ActiveFrom)

		if base.ActiveTo != nil && base.ActiveTo.Before(endTime) {
			// Phase Cadence overrides item cadence in all cases
			endTime = *base.ActiveTo
		}

		end = &endTime
	}

	return models.CadencedModel{
		ActiveFrom: start,
		ActiveTo:   end,
	}
}
