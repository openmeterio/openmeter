package subscription

import (
	"encoding/json"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

type SubscriptionItem struct {
	models.NamespacedID  `json:",inline"`
	models.ManagedModel  `json:",inline"`
	models.MetadataModel `json:",inline"`

	Annotations models.Annotations `json:"annotations"`

	// SubscriptionItem doesn't have a separate Cadence, only one relative to the phase, denoting if it's intentionally different from the phase's cadence.
	// The durations are relative to phase start.
	ActiveFromOverrideRelativeToPhaseStart *datetime.ISODuration `json:"activeFromOverrideRelativeToPhaseStart,omitempty"`
	ActiveToOverrideRelativeToPhaseStart   *datetime.ISODuration `json:"activeToOverrideRelativeToPhaseStart,omitempty"`

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
	type itemAlias SubscriptionItem

	var item itemAlias
	serde := struct {
		*itemAlias
		RateCard json.RawMessage `json:"rateCard"`
	}{
		itemAlias: &item,
	}

	if err := json.Unmarshal(b, &serde); err != nil {
		return fmt.Errorf("failed to JSON deserialize SubscriptionItem: %w", err)
	}

	rateCard, err := unmarshalSubscriptionItemRateCard(serde.RateCard)
	if err != nil {
		return err
	}

	item.RateCard = rateCard
	*i = SubscriptionItem(item)

	return nil
}

func unmarshalSubscriptionItemRateCard(data []byte) (productcatalog.RateCard, error) {
	var serde productcatalog.RateCardSerde
	if err := json.Unmarshal(data, &serde); err != nil {
		return nil, fmt.Errorf("failed to JSON deserialize SubscriptionItem rate card type: %w", err)
	}

	var rateCard productcatalog.RateCard
	switch serde.Type {
	case productcatalog.FlatFeeRateCardType:
		rateCard = &productcatalog.FlatFeeRateCard{}
	case productcatalog.UsageBasedRateCardType:
		rateCard = &productcatalog.UsageBasedRateCard{}
	default:
		return nil, fmt.Errorf("invalid RateCard type: %s", serde.Type)
	}

	if err := json.Unmarshal(data, rateCard); err != nil {
		return nil, fmt.Errorf("failed to JSON deserialize SubscriptionItem rate card: %w", err)
	}

	return rateCard, nil
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
