package subscription

import (
	"encoding/json"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
)

type SubscriptionItem struct {
	models.NamespacedID  `json:",inline"`
	models.ManagedModel  `json:",inline"`
	models.MetadataModel `json:",inline"`

	Annotations models.Annotations `json:"annotations"`

	// SubscriptionItem doesn't have a separate Cadence, only one relative to the phase, denoting if it's intentionally different from the phase's cadence.
	// The durations are relative to phase start.
	ActiveFromOverrideRelativeToPhaseStart *isodate.Period `json:"activeFromOverrideRelativeToPhaseStart,omitempty"`
	ActiveToOverrideRelativeToPhaseStart   *isodate.Period `json:"activeToOverrideRelativeToPhaseStart,omitempty"`

	// The defacto cadence of the item is calculated and persisted after each change.
	models.CadencedModel `json:",inline"`

	BillingBehaviorOverride BillingBehaviorOverride `json:"billingBehaviorOverride"`

	// SubscriptionID is the ID of the subscription this item belongs to.
	SubscriptionId string `json:"subscriptionId"`
	// PhaseID is the ID of the phase this item belongs to.
	PhaseId string `json:"phaseId"`
	// Key is the unique key of the item in the phase.
	Key string `json:"itemKey"`

	RateCard productcatalog.RateCard `json:"rateCard"`

	EntitlementID *string `json:"entitlementId,omitempty"`
	// Name
	Name string `json:"name"`

	// Description
	Description *string `json:"description,omitempty"`
}

func (i *SubscriptionItem) UnmarshalJSON(b []byte) error {
	// First unmarshal the type information to determine which concrete type to use
	var serdeTyp struct {
		RateCard productcatalog.RateCardSerde `json:"rateCard"`
	}

	if err := json.Unmarshal(b, &serdeTyp); err != nil {
		return fmt.Errorf("failed to JSON deserialize SubscriptionItemSpec: %w", err)
	}

	// Create a temporary struct with the correct concrete type for RateCard
	serde := struct {
		models.NamespacedID  `json:",inline"`
		models.ManagedModel  `json:",inline"`
		models.MetadataModel `json:",inline"`

		ActiveFromOverrideRelativeToPhaseStart *isodate.Period `json:"activeFromOverrideRelativeToPhaseStart,omitempty"`
		ActiveToOverrideRelativeToPhaseStart   *isodate.Period `json:"activeToOverrideRelativeToPhaseStart,omitempty"`

		models.CadencedModel `json:",inline"`

		BillingBehaviorOverride BillingBehaviorOverride `json:"billingBehaviorOverride"`

		SubscriptionId string `json:"subscriptionId"`
		PhaseId        string `json:"phaseId"`
		Key            string `json:"itemKey"`

		RateCard productcatalog.RateCard `json:"rateCard"`

		EntitlementID *string `json:"entitlementId,omitempty"`
		Name          string  `json:"name"`
		Description   *string `json:"description,omitempty"`
	}{
		RateCard: i.RateCard,
	}

	// Set the concrete type based on the type field
	switch serdeTyp.RateCard.Type {
	case productcatalog.FlatFeeRateCardType:
		serde.RateCard = &productcatalog.FlatFeeRateCard{}
	case productcatalog.UsageBasedRateCardType:
		serde.RateCard = &productcatalog.UsageBasedRateCard{}
	default:
		return fmt.Errorf("invalid RateCard type: %s", serdeTyp.RateCard.Type)
	}

	// Unmarshal the full object
	if err := json.Unmarshal(b, &serde); err != nil {
		return fmt.Errorf("failed to JSON deserialize SubscriptionItem: %w", err)
	}

	// Copy all fields from the temporary struct to the actual struct
	i.NamespacedID = serde.NamespacedID
	i.ManagedModel = serde.ManagedModel
	i.MetadataModel = serde.MetadataModel
	i.ActiveFromOverrideRelativeToPhaseStart = serde.ActiveFromOverrideRelativeToPhaseStart
	i.ActiveToOverrideRelativeToPhaseStart = serde.ActiveToOverrideRelativeToPhaseStart
	i.CadencedModel = serde.CadencedModel
	i.BillingBehaviorOverride = serde.BillingBehaviorOverride
	i.SubscriptionId = serde.SubscriptionId
	i.PhaseId = serde.PhaseId
	i.Key = serde.Key
	i.RateCard = serde.RateCard
	i.EntitlementID = serde.EntitlementID
	i.Name = serde.Name
	i.Description = serde.Description

	return nil
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
		MetadataModel:                          i.MetadataModel,
		Annotations:                            i.Annotations,
		CadencedModel:                          i.CadencedModel,
		ActiveFromOverrideRelativeToPhaseStart: i.ActiveFromOverrideRelativeToPhaseStart,
		ActiveToOverrideRelativeToPhaseStart:   i.ActiveToOverrideRelativeToPhaseStart,
		PhaseID:                                i.PhaseId,
		Key:                                    i.Key,
		RateCard:                               i.RateCard,
		EntitlementID:                          i.EntitlementID,
		Name:                                   i.Name,
		Description:                            i.Description,
		BillingBehaviorOverride:                i.BillingBehaviorOverride,
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
