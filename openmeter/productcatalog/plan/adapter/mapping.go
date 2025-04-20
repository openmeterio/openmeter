package adapter

import (
	"errors"
	"fmt"

	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/pkg/models"
)

func FromPlanRow(p entdb.Plan) (*plan.Plan, error) {
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

	// Check whether the addons were loaded or not.
	addons, err := p.Edges.AddonsOrErr()
	if err != nil {
		// Set addons to nil signaling that the addons were not loaded.
		pp.Addons = nil
	} else {
		planAddons := make([]plan.Addon, 0, len(addons))

		for _, addon := range addons {
			if addon == nil {
				continue
			}

			planAddon, err := FromPlanAddonRow(*addon)
			if err != nil {
				return nil, fmt.Errorf("invalid plan add-on assignment %s: %w", addon.ID, err)
			}

			planAddons = append(planAddons, *planAddon)
		}

		pp.Addons = &planAddons
	}

	if err := pp.Validate(); err != nil {
		return nil, fmt.Errorf("invalid plan %s: %w", pp.ID, err)
	}

	return pp, nil
}

func FromPlanAddonRow(a entdb.PlanAddon) (*plan.Addon, error) {
	planAddon := &plan.Addon{
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

	// Set Addon

	addon, err := a.Edges.AddonOrErr()
	if err != nil {
		return nil, errors.New("failed to cast add-on: add-on is not loaded")
	}

	aa, err := FromAddonRow(*addon)
	if err != nil {
		return nil, fmt.Errorf("failed to cast add-on: %w", err)
	}

	planAddon.Addon = *aa

	if err := planAddon.Validate(); err != nil {
		return nil, fmt.Errorf("invalid add-on [namespace=%s id=%s]: %w", planAddon.Namespace, planAddon.ID, err)
	}

	return planAddon, nil
}

func FromAddonRow(a entdb.Addon) (*productcatalog.Addon, error) {
	aa := &productcatalog.Addon{
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
		aa.RateCards = make(productcatalog.RateCards, 0, len(a.Edges.Ratecards))
		for _, edge := range a.Edges.Ratecards {
			if edge == nil {
				continue
			}

			ratecard, err := FromAddonRateCardRow(*edge)
			if err != nil {
				return nil, fmt.Errorf("invalid ratecard [namespace=%s key=%s]: %w", a.Namespace, edge.Key, err)
			}

			aa.RateCards = append(aa.RateCards, ratecard)
		}
	}

	if err := aa.Validate(); err != nil {
		return nil, fmt.Errorf("invalid add-on [namespace=%s key=%s]: %w", a.Namespace, aa.Key, err)
	}

	return aa, nil
}

func FromAddonRateCardRow(r entdb.AddonRateCard) (productcatalog.RateCard, error) {
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
		return nil, fmt.Errorf("invalid ratecard [namespace=%s key=%s]: invalid type %s: %w", r.Namespace, r.Key, r.Type, err)
	}

	if err = ratecard.Validate(); err != nil {
		return nil, fmt.Errorf("invalid ratecard [namespace=%s key=%s]: %w", r.Namespace, r.Key, err)
	}

	return ratecard, nil
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
		ratecard = &plan.RateCard{
			RateCardManagedFields: managed,
			RateCard: &productcatalog.FlatFeeRateCard{
				RateCardMeta:   meta,
				BillingCadence: billingCadence,
			},
		}
	case productcatalog.UsageBasedRateCardType:
		ratecard = &plan.RateCard{
			RateCardManagedFields: managed,
			RateCard: &productcatalog.UsageBasedRateCard{
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
		Discounts:           lo.EmptyableToPtr(meta.Discounts),
	}

	if managed, ok := r.(plan.ManagedRateCard); ok {
		managedFields := managed.ManagedFields()
		ratecard.Namespace = managedFields.Namespace
		ratecard.ID = managedFields.ID
		ratecard.PhaseID = managedFields.PhaseID
	}

	ratecard.FeatureKey = meta.FeatureKey
	ratecard.FeatureID = meta.FeatureID

	ratecard.BillingCadence = r.GetBillingCadence().ISOStringPtrOrNil()

	return ratecard, nil
}
