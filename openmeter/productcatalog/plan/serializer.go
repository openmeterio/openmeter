package plan

import (
	"encoding/json"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/isodate"
)

type (
	planAlias  Plan
	phaseAlias Phase
)

type rateCardAlias struct {
	Type                productcatalog.RateCardType `json:"type"`
	Key                 string                      `json:"key"`
	Name                string                      `json:"name"`
	Description         *string                     `json:"description,omitempty"`
	Metadata            map[string]string           `json:"metadata,omitempty"`
	FeatureKey          *string                     `json:"featureKey,omitempty"`
	FeatureID           *string                     `json:"featureID,omitempty"`
	BillingCadence      *string                     `json:"billingCadence,omitempty"`
	Price               json.RawMessage             `json:"price,omitempty"`
	EntitlementTemplate json.RawMessage             `json:"entitlementTemplate,omitempty"`
}

// MarshalJSON implements json.Marshaler
func (p Plan) MarshalJSON() ([]byte, error) {
	// Create a copy of the plan to avoid recursion
	plan := planAlias(p)

	// Convert phases to a format suitable for JSON
	phases := make([]json.RawMessage, len(p.Phases))
	for i, phase := range p.Phases {
		// Create a copy of the phase
		phaseJSON := phaseAlias(phase)

		// Convert rate cards to a format suitable for JSON
		rateCards := make([]rateCardAlias, len(phase.RateCards))
		for j, rc := range phase.RateCards {
			meta := rc.AsMeta()

			// Marshal price to raw JSON if present
			var priceJSON json.RawMessage
			if meta.Price != nil {
				priceBytes, err := json.Marshal(meta.Price)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal price: %w", err)
				}
				priceJSON = priceBytes
			}

			// Marshal entitlement template to raw JSON if present
			var entitlementTemplateJSON json.RawMessage
			if meta.EntitlementTemplate != nil {
				entitlementTemplateBytes, err := json.Marshal(meta.EntitlementTemplate)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal entitlement template: %w", err)
				}
				entitlementTemplateJSON = entitlementTemplateBytes
			}

			// Create rate card JSON
			rateCard := rateCardAlias{
				Type:                rc.Type(),
				Key:                 rc.Key(),
				Name:                meta.Name,
				Description:         meta.Description,
				Metadata:            meta.Metadata,
				FeatureKey:          meta.FeatureKey,
				FeatureID:           meta.FeatureID,
				Price:               priceJSON,
				EntitlementTemplate: entitlementTemplateJSON,
			}

			if bc := rc.GetBillingCadence(); bc != nil {
				bcStr := bc.String()
				rateCard.BillingCadence = &bcStr
			}

			rateCards[j] = rateCard
		}

		// Marshal phase with rate cards
		phaseBytes, err := json.Marshal(struct {
			phaseAlias
			RateCards []rateCardAlias `json:"rateCards"`
		}{
			phaseAlias: phaseJSON,
			RateCards:  rateCards,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to marshal phase: %w", err)
		}

		phases[i] = phaseBytes
	}

	// Marshal plan with phases
	return json.Marshal(struct {
		planAlias
		Phases []json.RawMessage `json:"phases"`
	}{
		planAlias: plan,
		Phases:    phases,
	})
}

// UnmarshalJSON implements json.Unmarshaler
func (p *Plan) UnmarshalJSON(data []byte) error {
	type planAlias Plan
	type phaseAlias Phase
	type rateCardAlias struct {
		Type                productcatalog.RateCardType `json:"type"`
		Key                 string                      `json:"key"`
		Name                string                      `json:"name"`
		Description         *string                     `json:"description,omitempty"`
		Metadata            map[string]string           `json:"metadata,omitempty"`
		FeatureKey          *string                     `json:"featureKey,omitempty"`
		FeatureID           *string                     `json:"featureID,omitempty"`
		BillingCadence      *string                     `json:"billingCadence,omitempty"`
		Price               json.RawMessage             `json:"price,omitempty"`
		EntitlementTemplate json.RawMessage             `json:"entitlementTemplate,omitempty"`
	}

	// Unmarshal the base plan
	var planData struct {
		planAlias
		Phases []json.RawMessage `json:"phases"`
	}
	if err := json.Unmarshal(data, &planData); err != nil {
		return fmt.Errorf("failed to unmarshal plan: %w", err)
	}

	// Copy the base plan data
	*p = Plan(planData.planAlias)

	// Unmarshal phases
	p.Phases = make([]Phase, len(planData.Phases))
	for i, phaseData := range planData.Phases {
		var phaseWithRateCards struct {
			phaseAlias
			RateCards []rateCardAlias `json:"rateCards"`
		}
		if err := json.Unmarshal(phaseData, &phaseWithRateCards); err != nil {
			return fmt.Errorf("failed to unmarshal phase: %w", err)
		}

		// Copy the phase data
		phase := Phase(phaseWithRateCards.phaseAlias)
		phase.NamespacedID.Namespace = p.Namespace
		phase.NamespacedID.ID = phase.Key
		phase.PlanID = p.ID

		// Unmarshal rate cards
		phase.RateCards = make([]productcatalog.RateCard, len(phaseWithRateCards.RateCards))
		for j, rcData := range phaseWithRateCards.RateCards {
			var price *productcatalog.Price
			if len(rcData.Price) > 0 {
				price = &productcatalog.Price{}
				if err := json.Unmarshal(rcData.Price, price); err != nil {
					return fmt.Errorf("failed to unmarshal price: %w", err)
				}
			}

			var entitlementTemplate *productcatalog.EntitlementTemplate
			if len(rcData.EntitlementTemplate) > 0 {
				entitlementTemplate = &productcatalog.EntitlementTemplate{}
				if err := json.Unmarshal(rcData.EntitlementTemplate, entitlementTemplate); err != nil {
					return fmt.Errorf("failed to unmarshal entitlement template: %w", err)
				}
			}

			meta := productcatalog.RateCardMeta{
				Key:                 rcData.Key,
				Name:                rcData.Name,
				Description:         rcData.Description,
				Metadata:            rcData.Metadata,
				FeatureKey:          rcData.FeatureKey,
				FeatureID:           rcData.FeatureID,
				Price:               price,
				EntitlementTemplate: entitlementTemplate,
			}

			var rc productcatalog.RateCard
			switch rcData.Type {
			case productcatalog.FlatFeeRateCardType:
				frc := &productcatalog.FlatFeeRateCard{RateCardMeta: meta}
				if rcData.BillingCadence != nil {
					period, err := isodate.String(*rcData.BillingCadence).Parse()
					if err != nil {
						return fmt.Errorf("invalid billing cadence for rate card %q: %w", rcData.Key, err)
					}
					frc.BillingCadence = &period
				}
				rc = frc

			case productcatalog.UsageBasedRateCardType:
				urc := &productcatalog.UsageBasedRateCard{RateCardMeta: meta}
				if rcData.BillingCadence != nil {
					period, err := isodate.String(*rcData.BillingCadence).Parse()
					if err != nil {
						return fmt.Errorf("invalid billing cadence for rate card %q: %w", rcData.Key, err)
					}
					urc.BillingCadence = period
				}
				rc = urc

			default:
				return fmt.Errorf("unsupported rate card type: %s", rcData.Type)
			}

			phase.RateCards[j] = rc
		}

		p.Phases[i] = phase
	}

	return nil
}
