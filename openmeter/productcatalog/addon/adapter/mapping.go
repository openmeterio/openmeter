package adapter

import (
	"errors"
	"fmt"

	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/pkg/models"
)

func FromAddonRow(a entdb.Addon) (*addon.Addon, error) {
	aa := &addon.Addon{
		NamespacedID: models.NamespacedID{
			Namespace: a.Namespace,
			ID:        a.ID,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: a.CreatedAt,
			UpdatedAt: a.UpdatedAt,
			DeletedAt: a.DeletedAt,
		},
		AddonMeta: productcatalog.AddonMeta{
			Key:          a.Key,
			Name:         a.Name,
			Description:  a.Description,
			Metadata:     a.Metadata,
			Annotations:  a.Annotations,
			Version:      a.Version,
			Currency:     currency.Code(a.Currency),
			InstanceType: a.InstanceType,
			EffectivePeriod: productcatalog.EffectivePeriod{
				EffectiveFrom: a.EffectiveFrom,
				EffectiveTo:   a.EffectiveTo,
			},
		},
	}

	// Set Rate Cards

	if len(a.Edges.Ratecards) > 0 {
		aa.RateCards = make(addon.RateCards, 0, len(a.Edges.Ratecards))
		for _, edge := range a.Edges.Ratecards {
			if edge == nil {
				continue
			}

			ratecard, err := FromAddonRateCardRow(*edge)
			if err != nil {
				return nil, fmt.Errorf("invalid ratecard [namespace=%s key=%s]: %w", aa.Namespace, edge.Key, err)
			}

			aa.RateCards = append(aa.RateCards, *ratecard)
		}
	}

	plans, err := a.Edges.PlansOrErr()
	if err != nil {
		aa.Plans = nil
	} else {
		planAddons := make([]addon.Plan, 0, len(plans))

		for _, plan := range plans {
			if plan == nil {
				continue
			}

			planAddon, err := FromPlanAddonRow(*plan)
			if err != nil {
				return nil, fmt.Errorf("invalid plan add-on assignment %s: %w", plan.ID, err)
			}

			planAddons = append(planAddons, *planAddon)
		}

		aa.Plans = &planAddons
	}

	return aa, nil
}

func FromAddonRateCardRow(r entdb.AddonRateCard) (*addon.RateCard, error) {
	meta := productcatalog.RateCardMeta{
		Key:                 r.Key,
		Name:                r.Name,
		Description:         r.Description,
		Metadata:            r.Metadata,
		EntitlementTemplate: r.EntitlementTemplate,
		FeatureKey:          r.FeatureKey,
		FeatureID:           r.FeatureID,
		TaxConfig:           r.TaxConfig,
		Price:               r.Price,
		Discounts:           lo.FromPtr(r.Discounts),
	}

	// Get billing cadence

	billingCadence, err := r.BillingCadence.ParsePtrOrNil()
	if err != nil {
		return nil, fmt.Errorf("invalid ratecard [namespace=%s key=%s]: billing cadence: %w", r.Namespace, r.Key, err)
	}

	// Managed fields

	managed := addon.RateCardManagedFields{
		ManagedModel: models.ManagedModel{
			CreatedAt: r.CreatedAt,
			UpdatedAt: r.UpdatedAt,
			DeletedAt: r.DeletedAt,
		},
		NamespacedID: models.NamespacedID{
			Namespace: r.Namespace,
			ID:        r.ID,
		},
		AddonID: r.AddonID,
	}

	var ratecard *addon.RateCard

	switch r.Type {
	case productcatalog.FlatFeeRateCardType:
		ratecard = &addon.RateCard{
			RateCardManagedFields: managed,
			RateCard: &productcatalog.FlatFeeRateCard{
				RateCardMeta:   meta,
				BillingCadence: billingCadence,
			},
		}
	case productcatalog.UsageBasedRateCardType:
		ratecard = &addon.RateCard{
			RateCardManagedFields: managed,
			RateCard: &productcatalog.UsageBasedRateCard{
				RateCardMeta:   meta,
				BillingCadence: lo.FromPtr(billingCadence),
			},
		}
	default:
		return nil, fmt.Errorf("invalid ratecard [namespace=%s key=%s]: invalid type %s: %w", r.Namespace, r.Key, r.Type, err)
	}

	return ratecard, nil
}

func FromPlanAddonRow(a entdb.PlanAddon) (*addon.Plan, error) {
	planAddon := &addon.Plan{
		NamespacedID: models.NamespacedID{
			Namespace: a.Namespace,
			ID:        a.ID,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: a.CreatedAt,
			UpdatedAt: a.UpdatedAt,
			DeletedAt: a.DeletedAt,
		},
		PlanAddonMeta: productcatalog.PlanAddonMeta{
			Metadata:    a.Metadata,
			Annotations: a.Annotations,
			PlanAddonConfig: productcatalog.PlanAddonConfig{
				FromPlanPhase: a.FromPlanPhase,
				MaxQuantity:   a.MaxQuantity,
			},
		},
	}

	// Set Plan

	plan, err := a.Edges.PlanOrErr()
	if err != nil {
		return nil, errors.New("failed to cast plan: plan is not loaded")
	}

	pp, err := FromPlanRow(*plan)
	if err != nil {
		return nil, fmt.Errorf("failed to cast add-on: %w", err)
	}

	planAddon.Plan = *pp

	return planAddon, nil
}

func FromPlanRow(p entdb.Plan) (*productcatalog.Plan, error) {
	billingCadence, err := p.BillingCadence.Parse()
	if err != nil {
		return nil, fmt.Errorf("invalid billing cadence %s: %w", p.BillingCadence, err)
	}

	pp := &productcatalog.Plan{
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
			BillingCadence:  billingCadence,
			ProRatingConfig: p.ProRatingConfig,
		},
	}

	if len(p.Edges.Phases) > 0 {
		phases := make([]productcatalog.Phase, len(p.Edges.Phases))
		for _, edge := range p.Edges.Phases {
			if edge == nil {
				continue
			}

			phase, err := FromPlanPhaseRow(*edge)
			if err != nil {
				return nil, fmt.Errorf("invalid phase %s: %w", edge.ID, err)
			}

			phases[edge.Index] = *phase
		}

		if len(phases) > 0 {
			pp.Phases = phases
		}
	}

	return pp, nil
}

func FromPlanPhaseRow(p entdb.PlanPhase) (*productcatalog.Phase, error) {
	pp := &productcatalog.Phase{
		PhaseMeta: productcatalog.PhaseMeta{
			Key:         p.Key,
			Name:        p.Name,
			Description: p.Description,
			Metadata:    p.Metadata,
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

			ratecard, err := FromPlanRateCardRow(*edge)
			if err != nil {
				return nil, fmt.Errorf("invalid rate card %s: %w", edge.ID, err)
			}

			pp.RateCards = append(pp.RateCards, ratecard)
		}
	}

	return pp, nil
}

func FromPlanRateCardRow(r entdb.PlanRateCard) (productcatalog.RateCard, error) {
	meta := productcatalog.RateCardMeta{
		Key:                 r.Key,
		Name:                r.Name,
		Description:         r.Description,
		Metadata:            r.Metadata,
		FeatureID:           r.FeatureID,
		FeatureKey:          r.FeatureKey,
		EntitlementTemplate: r.EntitlementTemplate,
		TaxConfig:           r.TaxConfig,
		Price:               r.Price,
		Discounts:           lo.FromPtr(r.Discounts),
	}

	// Get billing cadence

	billingCadence, err := r.BillingCadence.ParsePtrOrNil()
	if err != nil {
		return nil, fmt.Errorf("invalid rate card billing cadence %s: %w", r.ID, err)
	}

	var ratecard productcatalog.RateCard

	switch r.Type {
	case productcatalog.FlatFeeRateCardType:
		ratecard = &productcatalog.FlatFeeRateCard{
			RateCardMeta:   meta,
			BillingCadence: billingCadence,
		}
	case productcatalog.UsageBasedRateCardType:
		ratecard = &productcatalog.UsageBasedRateCard{
			RateCardMeta:   meta,
			BillingCadence: lo.FromPtr(billingCadence),
		}
	default:
		return nil, fmt.Errorf("invalid RateCard type %s: %w", r.Type, err)
	}

	return ratecard, nil
}

func asAddonRateCardRow(r productcatalog.RateCard) (entdb.AddonRateCard, error) {
	meta := r.AsMeta()

	ratecard := entdb.AddonRateCard{
		Key:                 meta.Key,
		Metadata:            meta.Metadata,
		Name:                meta.Name,
		Description:         meta.Description,
		EntitlementTemplate: meta.EntitlementTemplate,
		TaxConfig:           meta.TaxConfig,
		FeatureKey:          meta.FeatureKey,
		FeatureID:           meta.FeatureID,
		Price:               meta.Price,
		Type:                r.Type(),
		Discounts:           lo.EmptyableToPtr(meta.Discounts),
	}

	if managed, ok := r.(addon.ManagedRateCard); ok {
		managedFields := managed.ManagedFields()
		ratecard.Namespace = managedFields.Namespace
		ratecard.ID = managedFields.ID
	}

	switch v := r.(type) {
	case *productcatalog.FlatFeeRateCard:
		ratecard.BillingCadence = v.BillingCadence.ISOStringPtrOrNil()
	case *productcatalog.UsageBasedRateCard:
		ratecard.BillingCadence = v.BillingCadence.ISOStringPtrOrNil()
	default:
		return ratecard, fmt.Errorf("invalid ratecard [key=%s]: invalid type: %T", r.Key(), r)
	}

	return ratecard, nil
}
