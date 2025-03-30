package plan

import (
	"encoding/json"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
)

// PlanSerializer is a serialization/deserialization helper for Plan
type PlanSerializer struct {
	models.NamespacedID
	models.ManagedModel

	productcatalog.PlanMeta

	// Phases
	Phases []PhaseSerializer `json:"phases"`
}

// PhaseSerializer is a serialization/deserialization helper for Phase
type PhaseSerializer struct {
	models.ManagedModel

	productcatalog.PhaseMeta

	// Discounts stores a set of discount(s) applied to all or specific RateCards.
	Discounts productcatalog.Discounts `json:"discounts,omitempty"`

	// RateCards
	RateCards []RateCardSerializer `json:"rateCards"`
}

// RateCardSerializer is a serialization/deserialization helper for RateCard
type RateCardSerializer struct {
	Type productcatalog.RateCardType `json:"type"`

	// Common fields
	Key         string           `json:"key"`
	Name        string           `json:"name"`
	Description *string          `json:"description,omitempty"`
	Metadata    models.Metadata  `json:"metadata,omitempty"`
	Feature     *feature.Feature `json:"feature,omitempty"`

	// Type-specific fields
	BillingCadence      *string                             `json:"billingCadence,omitempty"`
	Price               *productcatalog.Price               `json:"price,omitempty"`
	EntitlementTemplate *productcatalog.EntitlementTemplate `json:"entitlementTemplate,omitempty"`
}

// MarshalJSON implements json.Marshaler
func (p Plan) MarshalJSON() ([]byte, error) {
	serde := PlanSerializer{
		NamespacedID: p.NamespacedID,
		ManagedModel: p.ManagedModel,
		PlanMeta:     p.PlanMeta,
		Phases:       make([]PhaseSerializer, len(p.Phases)),
	}

	for i, phase := range p.Phases {
		phaseSerde := PhaseSerializer{
			ManagedModel: phase.ManagedModel,
			PhaseMeta:    phase.PhaseMeta,
			Discounts:    phase.Discounts,
			RateCards:    make([]RateCardSerializer, len(phase.RateCards)),
		}

		for j, rc := range phase.RateCards {
			meta := rc.AsMeta()

			rcSerde := RateCardSerializer{
				Type:                rc.Type(),
				Key:                 rc.Key(),
				Name:                meta.Name,
				Description:         meta.Description,
				Metadata:            meta.Metadata,
				Feature:             rc.Feature(),
				Price:               meta.Price,
				EntitlementTemplate: meta.EntitlementTemplate,
			}

			if bc := rc.GetBillingCadence(); bc != nil {
				bcStr := bc.String()
				rcSerde.BillingCadence = &bcStr
			}

			phaseSerde.RateCards[j] = rcSerde
		}

		serde.Phases[i] = phaseSerde
	}

	return json.Marshal(serde)
}

// UnmarshalJSON implements json.Unmarshaler
func (p *Plan) UnmarshalJSON(data []byte) error {
	var serde PlanSerializer
	if err := json.Unmarshal(data, &serde); err != nil {
		return fmt.Errorf("failed to unmarshal plan: %w", err)
	}

	p.NamespacedID = serde.NamespacedID
	p.ManagedModel = serde.ManagedModel
	p.PlanMeta = serde.PlanMeta
	p.Phases = make([]Phase, len(serde.Phases))

	for i, phaseSerde := range serde.Phases {
		phase := Phase{
			PhaseManagedFields: PhaseManagedFields{
				ManagedModel: phaseSerde.ManagedModel,
				NamespacedID: models.NamespacedID{
					Namespace: p.Namespace,
					ID:        phaseSerde.Key,
				},
				PlanID: p.ID,
			},
			Phase: productcatalog.Phase{
				PhaseMeta: phaseSerde.PhaseMeta,
				Discounts: phaseSerde.Discounts,
				RateCards: make([]productcatalog.RateCard, len(phaseSerde.RateCards)),
			},
		}

		for j, rcSerde := range phaseSerde.RateCards {
			var rc productcatalog.RateCard

			switch rcSerde.Type {
			case productcatalog.FlatFeeRateCardType:
				frc := &productcatalog.FlatFeeRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:                 rcSerde.Key,
						Name:                rcSerde.Name,
						Description:         rcSerde.Description,
						Metadata:            rcSerde.Metadata,
						Feature:             rcSerde.Feature,
						Price:               rcSerde.Price,
						EntitlementTemplate: rcSerde.EntitlementTemplate,
					},
				}
				if rcSerde.BillingCadence != nil {
					period, err := isodate.String(*rcSerde.BillingCadence).Parse()
					if err != nil {
						return fmt.Errorf("invalid billing cadence for rate card %q: %w", rcSerde.Key, err)
					}
					frc.BillingCadence = &period
				}
				rc = frc

			case productcatalog.UsageBasedRateCardType:
				urc := &productcatalog.UsageBasedRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:                 rcSerde.Key,
						Name:                rcSerde.Name,
						Description:         rcSerde.Description,
						Metadata:            rcSerde.Metadata,
						Feature:             rcSerde.Feature,
						Price:               rcSerde.Price,
						EntitlementTemplate: rcSerde.EntitlementTemplate,
					},
				}
				if rcSerde.BillingCadence != nil {
					period, err := isodate.String(*rcSerde.BillingCadence).Parse()
					if err != nil {
						return fmt.Errorf("invalid billing cadence for rate card %q: %w", rcSerde.Key, err)
					}
					urc.BillingCadence = period
				}
				rc = urc

			default:
				return fmt.Errorf("unsupported rate card type: %s", rcSerde.Type)
			}

			phase.RateCards[j] = rc
		}

		p.Phases[i] = phase
	}

	return nil
}
