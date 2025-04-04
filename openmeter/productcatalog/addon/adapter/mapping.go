package adapter

import (
	"fmt"

	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/models"
)

func fromAddonRow(a entdb.Addon) (*addon.Addon, error) {
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
			Key:         a.Key,
			Name:        a.Name,
			Description: a.Description,
			Metadata:    a.Metadata,
			Annotations: a.Annotations,
			Version:     a.Version,
			Currency:    currency.Code(a.Currency),
			EffectivePeriod: productcatalog.EffectivePeriod{
				EffectiveFrom: a.EffectiveFrom,
				EffectiveTo:   a.EffectiveTo,
			},
		},
	}

	// Set Rate Cards

	if len(a.Edges.Ratecards) > 0 {
		aa.RateCards = make([]productcatalog.RateCard, 0, len(a.Edges.Ratecards))
		for _, edge := range a.Edges.Ratecards {
			if edge == nil {
				continue
			}

			ratecard, err := fromAddonRateCardRow(*edge)
			if err != nil {
				return nil, fmt.Errorf("invalid ratecard [namespace=%s key=%s]: %w", aa.Namespace, edge.Key, err)
			}

			aa.RateCards = append(aa.RateCards, ratecard)
		}
	}

	if err := aa.Validate(); err != nil {
		return nil, fmt.Errorf("invalid add-on [namespace=%s key=%s]: %w", aa.Namespace, aa.Key, err)
	}

	return aa, nil
}

func fromAddonRateCardRow(r entdb.AddonRateCard) (productcatalog.RateCard, error) {
	meta := productcatalog.RateCardMeta{
		Key:                 r.Key,
		Name:                r.Name,
		Description:         r.Description,
		Metadata:            r.Metadata,
		EntitlementTemplate: r.EntitlementTemplate,
		TaxConfig:           r.TaxConfig,
		Price:               r.Price,
		Discounts:           lo.FromPtrOr(r.Discounts, productcatalog.Discounts{}),
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

	var ratecard productcatalog.RateCard

	switch r.Type {
	case productcatalog.FlatFeeRateCardType:
		ratecard = &addon.FlatFeeRateCard{
			RateCardManagedFields: managed,
			FlatFeeRateCard: productcatalog.FlatFeeRateCard{
				RateCardMeta:   meta,
				BillingCadence: billingCadence,
			},
		}
	case productcatalog.UsageBasedRateCardType:
		ratecard = &addon.UsageBasedRateCard{
			RateCardManagedFields: managed,
			UsageBasedRateCard: productcatalog.UsageBasedRateCard{
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
		Price:               meta.Price,
		Type:                r.Type(),
		Discounts:           lo.EmptyableToPtr(meta.Discounts),
	}

	if managed, ok := r.(addon.ManagedRateCard); ok {
		managedFields := managed.ManagedFields()
		ratecard.Namespace = managedFields.Namespace
		ratecard.ID = managedFields.ID
	}

	if meta.Feature != nil {
		ratecard.FeatureKey = &meta.Feature.Key
		ratecard.FeatureID = &meta.Feature.ID
	}

	switch v := r.(type) {
	case *productcatalog.FlatFeeRateCard:
		ratecard.BillingCadence = v.BillingCadence.ISOStringPtrOrNil()
	case *addon.FlatFeeRateCard:
		ratecard.BillingCadence = v.FlatFeeRateCard.BillingCadence.ISOStringPtrOrNil()
	case *productcatalog.UsageBasedRateCard:
		ratecard.BillingCadence = v.BillingCadence.ISOStringPtrOrNil()
	case *addon.UsageBasedRateCard:
		ratecard.BillingCadence = v.UsageBasedRateCard.BillingCadence.ISOStringPtrOrNil()
	default:
		return ratecard, fmt.Errorf("invalid ratecard [key=%s]: invalid type: %T", r.Key(), r)
	}

	return ratecard, nil
}
