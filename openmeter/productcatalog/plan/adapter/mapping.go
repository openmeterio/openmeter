package adapter

import (
	"fmt"

	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	productcatalogmodel "github.com/openmeterio/openmeter/openmeter/productcatalog/model"
	planentity "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/entity"
	"github.com/openmeterio/openmeter/pkg/datex"
	"github.com/openmeterio/openmeter/pkg/models"
)

func fromPlanRow(p entdb.Plan) (*planentity.Plan, error) {
	conf := planentity.NewPlanConfig{
		NamespacedID: models.NamespacedID{
			Namespace: p.Namespace,
			ID:        p.ID,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
			DeletedAt: p.DeletedAt,
		},
		Plan: productcatalogmodel.PlanGeneric{
			Key:         p.Key,
			Name:        p.Name,
			Description: p.Description,
			Metadata:    p.Metadata,
			Version:     p.Version,
			Currency:    currency.Code(p.Currency),
			EffectivePeriod: productcatalogmodel.EffectivePeriod{
				EffectiveFrom: p.EffectiveFrom,
				EffectiveTo:   p.EffectiveTo,
			},
		},
	}

	if len(p.Edges.Phases) > 0 {
		phases := make([]planentity.Phase, 0, len(p.Edges.Phases))
		for _, edge := range p.Edges.Phases {
			if edge == nil {
				continue
			}

			phase, err := fromPlanPhaseRow(*edge)
			if err != nil {
				return nil, fmt.Errorf("invalid phase %s: %w", edge.ID, err)
			}

			phases = append(phases, *phase)
		}

		if len(phases) > 0 {
			conf.Phases = phases
		}
	}

	pp := planentity.NewPlan(conf)

	if err := pp.Plan.Validate(); err != nil {
		return nil, fmt.Errorf("invalid plan %s: %w", pp.ID, err)
	}

	return &pp, nil
}

func fromPlanPhaseRow(p entdb.PlanPhase) (*planentity.Phase, error) {
	conf := planentity.NewPhaseConfig{
		NamespacedID: models.NamespacedID{
			Namespace: p.Namespace,
			ID:        p.ID,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
			DeletedAt: p.DeletedAt,
		},
		PhaseGeneric: productcatalogmodel.PhaseGeneric{
			Key:         p.Key,
			Name:        p.Name,
			Description: p.Description,
			Metadata:    p.Metadata,
			PlanID:      p.PlanID,
		},
	}

	// Set Interval

	startAfter, err := p.StartAfter.Parse()
	if err != nil {
		return nil, fmt.Errorf("invalid startAfter %v: %w", p.StartAfter, err)
	}

	conf.PhaseGeneric.StartAfter = startAfter

	// Set Rate Cards

	if len(p.Edges.Ratecards) > 0 {
		ratecards := make([]planentity.RateCard, 0, len(p.Edges.Ratecards))
		for _, edge := range p.Edges.Ratecards {
			if edge == nil {
				continue
			}

			ratecard, err := fromPlanRateCardRow(*edge)
			if err != nil {
				return nil, fmt.Errorf("invalid rate card %s: %w", edge.ID, err)
			}

			ratecards = append(ratecards, *ratecard)
		}

		if len(ratecards) > 0 {
			conf.RateCards = ratecards
		}
	}

	pp := planentity.NewPhase(conf)

	if err = pp.Phase.Validate(); err != nil {
		return nil, fmt.Errorf("invalid plan phase %s: %w", pp.ID, err)
	}

	return &pp, nil
}

func fromPlanRateCardRow(r entdb.PlanRateCard) (*planentity.RateCard, error) {
	nsId := models.NamespacedID{
		Namespace: r.Namespace,
		ID:        r.ID,
	}
	nsModel := models.ManagedModel{
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
		DeletedAt: r.DeletedAt,
	}
	meta := productcatalogmodel.RateCardMeta{
		Key:                 r.Key,
		Type:                r.Type,
		Name:                r.Name,
		Description:         r.Description,
		Metadata:            r.Metadata,
		EntitlementTemplate: r.EntitlementTemplate,
		TaxConfig:           r.TaxConfig,
		PhaseID:             r.PhaseID,
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

	var ratecard productcatalogmodel.RateCard

	switch r.Type {
	case productcatalogmodel.FlatFeeRateCardType:
		ratecard = productcatalogmodel.NewRateCardFrom(productcatalogmodel.FlatFeeRateCard{
			RateCardMeta:   meta,
			BillingCadence: billingCadence,
			Price:          lo.FromPtrOr(r.Price, productcatalogmodel.Price{}),
		})
	case productcatalogmodel.UsageBasedRateCardType:
		ratecard = productcatalogmodel.NewRateCardFrom(productcatalogmodel.UsageBasedRateCard{
			RateCardMeta:   meta,
			BillingCadence: lo.FromPtrOr(billingCadence, datex.Period{}),
			Price:          r.Price,
		})
	}
	if err = ratecard.Validate(); err != nil {
		return nil, fmt.Errorf("invalid rate card %s: %w", r.ID, err)
	}

	return &planentity.RateCard{
		NamespacedID: nsId,
		ManagedModel: nsModel,
		RateCard:     ratecard,
	}, nil
}

func asPlanRateCardRow(r planentity.RateCard) (entdb.PlanRateCard, error) {
	meta, err := r.AsMeta()
	if err != nil {
		return entdb.PlanRateCard{}, fmt.Errorf("failed to cast rate card to meta: %w", err)
	}

	ratecard := entdb.PlanRateCard{
		Key:                 meta.Key,
		Metadata:            meta.Metadata,
		Name:                meta.Name,
		Description:         meta.Description,
		EntitlementTemplate: meta.EntitlementTemplate,
		TaxConfig:           meta.TaxConfig,
	}

	if meta.Feature != nil {
		ratecard.FeatureKey = &meta.Feature.Key
		ratecard.FeatureID = &meta.Feature.ID
	}

	switch r.Type() {
	case productcatalogmodel.FlatFeeRateCardType:
		flat, err := r.AsFlatFee()
		if err != nil {
			return entdb.PlanRateCard{}, fmt.Errorf("failed to cast flat fee rate card: %w", err)
		}

		ratecard.Type = productcatalogmodel.FlatFeeRateCardType
		ratecard.BillingCadence = flat.BillingCadence.ISOStringPtrOrNil()
		ratecard.Price = &flat.Price

	case productcatalogmodel.UsageBasedRateCardType:
		usage, err := r.AsUsageBased()
		if err != nil {
			return entdb.PlanRateCard{}, fmt.Errorf("failed to cast usage based rate card: %w", err)
		}

		ratecard.Type = productcatalogmodel.UsageBasedRateCardType
		ratecard.BillingCadence = lo.ToPtr(usage.BillingCadence.ISOString())
		ratecard.Price = usage.Price
	}

	return ratecard, nil
}
