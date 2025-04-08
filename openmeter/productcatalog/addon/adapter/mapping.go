package adapter

import (
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

	if err := aa.Validate(); err != nil {
		return nil, fmt.Errorf("invalid add-on [namespace=%s key=%s]: %w", aa.Namespace, aa.Key, err)
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
		Discounts:           lo.FromPtrOr(r.Discounts, productcatalog.Discounts{}),
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

	if err = ratecard.Validate(); err != nil {
		return nil, fmt.Errorf("invalid ratecard [namespace=%s key=%s]: %w", r.Namespace, r.Key, err)
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
