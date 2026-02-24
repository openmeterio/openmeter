package adapter

import (
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// MapChargeFromDB dispatches to the type-specific mapper based on the charge's type field.
func MapChargeFromDB(entity *entdb.Charge) (charges.Charge, error) {
	switch entity.Type {
	case charges.ChargeTypeFlatFee:
		ff, err := MapFlatFeeChargeFromDB(entity)
		if err != nil {
			return charges.Charge{}, fmt.Errorf("mapping flat fee charge [id=%s]: %w", entity.ID, err)
		}

		return ff.AsCharge(), nil
	case charges.ChargeTypeUsageBased:
		ub, err := MapUsageBasedChargeFromDB(entity)
		if err != nil {
			return charges.Charge{}, fmt.Errorf("mapping usage based charge [id=%s]: %w", entity.ID, err)
		}

		return ub.AsCharge(), nil
	case charges.ChargeTypeCreditPurchase:
		cp, err := MapCreditPurchaseChargeFromDB(entity)
		if err != nil {
			return charges.Charge{}, fmt.Errorf("mapping credit purchase charge [id=%s]: %w", entity.ID, err)
		}

		return cp.AsCharge(), nil
	default:
		return charges.Charge{}, fmt.Errorf("unknown charge type: %s", entity.Type)
	}
}

// MapFlatFeeChargeFromDB converts a DB Charge entity (with loaded FlatFee edge) to a FlatFeeCharge.
func MapFlatFeeChargeFromDB(entity *entdb.Charge) (charges.FlatFeeCharge, error) {
	if entity.Edges.FlatFee == nil {
		return charges.FlatFeeCharge{}, fmt.Errorf("flat_fee edge not loaded for charge [id=%s]", entity.ID)
	}

	ff := entity.Edges.FlatFee

	var percentageDiscounts *productcatalog.PercentageDiscount
	if ff.Discounts != nil {
		percentageDiscounts = ff.Discounts.Percentage
	}

	return charges.FlatFeeCharge{
		ManagedResource: mapManagedResourceFromDB(entity),
		Status:          entity.Status,
		Intent: charges.FlatFeeIntent{
			IntentMeta:            mapIntentMetaFromDB(entity),
			InvoiceAt:             ff.InvoiceAt.UTC(),
			SettlementMode:        ff.SettlementMode,
			PaymentTerm:           ff.PaymentTerm,
			FeatureKey:            lo.FromPtrOr(ff.FeatureKey, ""),
			PercentageDiscounts:   percentageDiscounts,
			ProRating:             proRatingConfigFromDB(ff.ProRating),
			AmountBeforeProration: ff.AmountBeforeProration,
			AmountAfterProration:  ff.AmountAfterProration,
		},
		State: charges.FlatFeeState{},
	}, nil
}

// MapUsageBasedChargeFromDB converts a DB Charge entity (with loaded UsageBased edge) to a UsageBasedCharge.
func MapUsageBasedChargeFromDB(entity *entdb.Charge) (charges.UsageBasedCharge, error) {
	if entity.Edges.UsageBased == nil {
		return charges.UsageBasedCharge{}, fmt.Errorf("usage_based edge not loaded for charge [id=%s]", entity.ID)
	}

	ub := entity.Edges.UsageBased

	if ub.Price == nil {
		return charges.UsageBasedCharge{}, fmt.Errorf("price is nil for usage based charge [id=%s]", entity.ID)
	}

	return charges.UsageBasedCharge{
		ManagedResource: mapManagedResourceFromDB(entity),
		Status:          entity.Status,
		Intent: charges.UsageBasedIntent{
			IntentMeta:     mapIntentMetaFromDB(entity),
			Price:          *ub.Price,
			FeatureKey:     ub.FeatureKey,
			InvoiceAt:      ub.InvoiceAt.UTC(),
			SettlementMode: ub.SettlementMode,
			Discounts:      ub.Discounts,
		},
		State: charges.UsageBasedState{},
	}, nil
}

// MapCreditPurchaseChargeFromDB converts a DB Charge entity (with loaded CreditPurchase edge) to a CreditPurchaseCharge.
func MapCreditPurchaseChargeFromDB(entity *entdb.Charge) (charges.CreditPurchaseCharge, error) {
	if entity.Edges.CreditPurchase == nil {
		return charges.CreditPurchaseCharge{}, fmt.Errorf("credit_purchase edge not loaded for charge [id=%s]", entity.ID)
	}

	cp := entity.Edges.CreditPurchase

	return charges.CreditPurchaseCharge{
		ManagedResource: mapManagedResourceFromDB(entity),
		Status:          entity.Status,
		Intent: charges.CreditPurchaseIntent{
			IntentMeta:   mapIntentMetaFromDB(entity),
			CreditAmount: cp.CreditAmount,
			Settlement:   cp.Settlement,
		},
		State: charges.CreditPurchaseState{
			Status: cp.Status,
		},
	}, nil
}

// mapManagedResourceFromDB extracts the ManagedResource from a DB Charge entity.
func mapManagedResourceFromDB(entity *entdb.Charge) models.ManagedResource {
	return models.NewManagedResource(models.ManagedResourceInput{
		ID:          entity.ID,
		Namespace:   entity.Namespace,
		Name:        entity.Name,
		Description: entity.Description,
		CreatedAt:   entity.CreatedAt,
		UpdatedAt:   entity.UpdatedAt,
		DeletedAt:   entity.DeletedAt,
	})
}

// mapIntentMetaFromDB extracts the IntentMeta from a DB Charge entity.
func mapIntentMetaFromDB(entity *entdb.Charge) charges.IntentMeta {
	return charges.IntentMeta{
		Metadata:    entity.Metadata,
		Annotations: entity.Annotations,
		ManagedBy:   entity.ManagedBy,
		CustomerID:  entity.CustomerID,
		Currency:    entity.Currency,
		ServicePeriod: timeutil.ClosedPeriod{
			From: entity.ServicePeriodFrom.UTC(),
			To:   entity.ServicePeriodTo.UTC(),
		},
		FullServicePeriod: timeutil.ClosedPeriod{
			From: entity.FullServicePeriodFrom.UTC(),
			To:   entity.FullServicePeriodTo.UTC(),
		},
		BillingPeriod: timeutil.ClosedPeriod{
			From: entity.BillingPeriodFrom.UTC(),
			To:   entity.BillingPeriodTo.UTC(),
		},
		UniqueReferenceID: entity.UniqueReferenceID,
		Subscription:      mapSubscriptionRefFromDB(entity),
	}
}

// mapSubscriptionRefFromDB extracts a SubscriptionReference from a DB Charge entity, returning nil if any ID is missing.
func mapSubscriptionRefFromDB(entity *entdb.Charge) *charges.SubscriptionReference {
	if entity.SubscriptionID == nil || entity.SubscriptionPhaseID == nil || entity.SubscriptionItemID == nil {
		return nil
	}

	return &charges.SubscriptionReference{
		SubscriptionID: *entity.SubscriptionID,
		PhaseID:        *entity.SubscriptionPhaseID,
		ItemID:         *entity.SubscriptionItemID,
	}
}

// proRatingConfigFromDB converts a DB ProRatingModeAdapterEnum to a ProRatingConfig.
func proRatingConfigFromDB(pr charges.ProRatingModeAdapterEnum) productcatalog.ProRatingConfig {
	switch pr {
	case charges.ProratePricesProratingAdapterMode:
		return productcatalog.ProRatingConfig{
			Enabled: true,
			Mode:    productcatalog.ProRatingModeProratePrices,
		}
	default:
		return productcatalog.ProRatingConfig{
			Enabled: false,
			Mode:    productcatalog.ProRatingModeProratePrices,
		}
	}
}

// proRatingConfigToDB converts a ProRatingConfig to a DB ProRatingModeAdapterEnum.
func proRatingConfigToDB(pc productcatalog.ProRatingConfig) (charges.ProRatingModeAdapterEnum, error) {
	if !pc.Enabled {
		return charges.NoProratingAdapterMode, nil
	}

	if pc.Mode == productcatalog.ProRatingModeProratePrices {
		return charges.ProratePricesProratingAdapterMode, nil
	}

	return "", fmt.Errorf("invalid pro rating mode: %s", pc.Mode)
}
