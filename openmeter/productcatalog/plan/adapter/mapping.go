package adapter

import (
	"fmt"

	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/pkg/models"
)

func fromPlanRow(p entdb.Plan) (*plan.Plan, error) {
	pp := &plan.Plan{
		NamespacedID: models.NamespacedID{
			Namespace: p.Namespace,
			ID:        p.ID,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
			DeletedAt: p.DeletedAt,
		},
		PlanMeta: productcatalog.PlanMeta{
			Key:         p.Key,
			Name:        p.Name,
			Description: p.Description,
			Metadata:    p.Metadata,
			Version:     p.Version,
			Currency:    currency.Code(p.Currency),
			EffectivePeriod: productcatalog.EffectivePeriod{
				EffectiveFrom: p.EffectiveFrom,
				EffectiveTo:   p.EffectiveTo,
			},
			Alignment: productcatalog.Alignment{
				BillablesMustAlign: p.BillablesMustAlign,
			},
		},
	}

	if len(p.Edges.Phases) > 0 {
		phases := make([]plan.Phase, len(p.Edges.Phases))
		for _, edge := range p.Edges.Phases {
			if edge == nil {
				continue
			}

			phase, err := fromPlanPhaseRow(*edge)
			if err != nil {
				return nil, fmt.Errorf("invalid phase %s: %w", edge.ID, err)
			}

			phases[edge.Index] = *phase
		}

		if len(phases) > 0 {
			pp.Phases = phases
		}
	}

	if err := pp.Validate(); err != nil {
		return nil, fmt.Errorf("invalid plan %s: %w", pp.ID, err)
	}

	return pp, nil
}

func fromPlanPhaseRow(p entdb.PlanPhase) (*plan.Phase, error) {
	pp := &plan.Phase{
		PhaseManagedFields: plan.PhaseManagedFields{
			ManagedModel: models.ManagedModel{
				CreatedAt: p.CreatedAt,
				UpdatedAt: p.UpdatedAt,
				DeletedAt: p.DeletedAt,
			},
			NamespacedID: models.NamespacedID{
				Namespace: p.Namespace,
				ID:        p.ID,
			},
			PlanID: p.PlanID,
		},
		Phase: productcatalog.Phase{
			PhaseMeta: productcatalog.PhaseMeta{
				Key:         p.Key,
				Name:        p.Name,
				Description: p.Description,
				Metadata:    p.Metadata,
			},
		},
	}

	// Set Interval

	duration, err := p.Duration.ParsePtrOrNil()
	if err != nil {
		return nil, fmt.Errorf("invalid duration %v: %w", p.Duration, err)
	}

	pp.Duration = duration

	// Set Rate Cards

	if len(p.Edges.Ratecards) > 0 {
		pp.RateCards = make([]productcatalog.RateCard, 0, len(p.Edges.Ratecards))
		for _, edge := range p.Edges.Ratecards {
			if edge == nil {
				continue
			}

			ratecard, err := fromPlanRateCardRow(*edge)
			if err != nil {
				return nil, fmt.Errorf("invalid rate card %s: %w", edge.ID, err)
			}

			pp.RateCards = append(pp.RateCards, ratecard)
		}
	}

	if err = pp.Validate(); err != nil {
		return nil, fmt.Errorf("invalid plan phase %s: %w", pp.ID, err)
	}

	return pp, nil
}

func fromPlanRateCardRow(r entdb.PlanRateCard) (productcatalog.RateCard, error) {
	meta := productcatalog.RateCardMeta{
		Key:                 r.Key,
		Name:                r.Name,
		Description:         r.Description,
		Metadata:            r.Metadata,
		EntitlementTemplate: r.EntitlementTemplate,
		TaxConfig:           r.TaxConfig,
		Price:               r.Price,
	}

	// Resolve feature

	if r.Edges.Features != nil {
		meta.Feature = &feature.Feature{
			Namespace:           r.Edges.Features.Namespace,
			ID:                  r.Edges.Features.ID,
			Name:                r.Edges.Features.Name,
			Key:                 r.Edges.Features.Key,
			MeterSlug:           r.Edges.Features.MeterSlug,
			MeterGroupByFilters: r.Edges.Features.MeterGroupByFilters,
			Metadata:            r.Edges.Features.Metadata,
			ArchivedAt:          r.Edges.Features.ArchivedAt,
			CreatedAt:           r.Edges.Features.CreatedAt,
			UpdatedAt:           r.Edges.Features.UpdatedAt,
		}
	}

	// Get billing cadence

	billingCadence, err := r.BillingCadence.ParsePtrOrNil()
	if err != nil {
		return nil, fmt.Errorf("invalid rate card billing cadence %s: %w", r.ID, err)
	}

	// Managed fields

	managed := plan.RateCardManagedFields{
		ManagedModel: models.ManagedModel{
			CreatedAt: r.CreatedAt,
			UpdatedAt: r.UpdatedAt,
			DeletedAt: r.DeletedAt,
		},
		NamespacedID: models.NamespacedID{
			Namespace: r.Namespace,
			ID:        r.ID,
		},
		PhaseID: r.PhaseID,
	}

	var ratecard productcatalog.RateCard

	switch r.Type {
	case productcatalog.FlatFeeRateCardType:
		ratecard = &plan.FlatFeeRateCard{
			RateCardManagedFields: managed,
			FlatFeeRateCard: productcatalog.FlatFeeRateCard{
				RateCardMeta:   meta,
				BillingCadence: billingCadence,
			},
		}
	case productcatalog.UsageBasedRateCardType:
		ratecard = &plan.UsageBasedRateCard{
			RateCardManagedFields: managed,
			UsageBasedRateCard: productcatalog.UsageBasedRateCard{
				RateCardMeta:   meta,
				BillingCadence: lo.FromPtr(billingCadence),
			},
		}
	default:
		return nil, fmt.Errorf("invalid RateCard type %s: %w", r.Type, err)
	}

	if err = ratecard.Validate(); err != nil {
		return nil, fmt.Errorf("invalid RateCard %s: %w", r.ID, err)
	}

	return ratecard, nil
}

func asPlanRateCardRow(r productcatalog.RateCard) (entdb.PlanRateCard, error) {
	meta := r.AsMeta()

	ratecard := entdb.PlanRateCard{
		Key:                 meta.Key,
		Metadata:            meta.Metadata,
		Name:                meta.Name,
		Description:         meta.Description,
		EntitlementTemplate: meta.EntitlementTemplate,
		TaxConfig:           meta.TaxConfig,
		Price:               meta.Price,
		Type:                r.Type(),
	}

	if managed, ok := r.(plan.ManagedRateCard); ok {
		managedFields := managed.ManagedFields()
		ratecard.Namespace = managedFields.Namespace
		ratecard.ID = managedFields.ID
		ratecard.PhaseID = managedFields.PhaseID
	}

	if meta.Feature != nil {
		ratecard.FeatureKey = &meta.Feature.Key
		ratecard.FeatureID = &meta.Feature.ID
	}

	switch v := r.(type) {
	case *productcatalog.FlatFeeRateCard:
		ratecard.BillingCadence = v.BillingCadence.ISOStringPtrOrNil()
	case *plan.FlatFeeRateCard:
		ratecard.BillingCadence = v.FlatFeeRateCard.BillingCadence.ISOStringPtrOrNil()
	case *productcatalog.UsageBasedRateCard:
		ratecard.BillingCadence = v.BillingCadence.ISOStringPtrOrNil()
	case *plan.UsageBasedRateCard:
		ratecard.BillingCadence = v.UsageBasedRateCard.BillingCadence.ISOStringPtrOrNil()
	default:
		return ratecard, fmt.Errorf("invalid RateCard type: %T", r)
	}

	return ratecard, nil
}
