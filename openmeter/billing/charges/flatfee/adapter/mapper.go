package adapter

import (
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/chargemeta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

// MapFlatFeeChargeFromDB converts a DB Charge entity (with loaded FlatFee edge) to a FlatFeeCharge.
func MapChargeFlatFeeFromDB(entity *entdb.ChargeFlatFee, expands meta.Expands) (flatfee.Charge, error) {
	var percentageDiscounts *productcatalog.PercentageDiscount
	if entity.Discounts != nil {
		percentageDiscounts = entity.Discounts.Percentage
	}

	mappedMeta := chargemeta.MapFromDB(entity)

	charge := flatfee.Charge{
		ManagedResource: mappedMeta.ManagedResource,
		Status:          mappedMeta.Status,
		State: flatfee.State{
			AdvanceAfter: mappedMeta.AdvanceAfter,
		},
		Intent: flatfee.Intent{
			Intent:                mappedMeta.Intent,
			InvoiceAt:             entity.InvoiceAt.UTC(),
			SettlementMode:        entity.SettlementMode,
			PaymentTerm:           entity.PaymentTerm,
			FeatureKey:            lo.FromPtrOr(entity.FeatureKey, ""),
			PercentageDiscounts:   percentageDiscounts,
			ProRating:             proRatingConfigFromDB(entity.ProRating),
			AmountBeforeProration: entity.AmountBeforeProration,
			AmountAfterProration:  entity.AmountAfterProration,
		},
	}

	if expands.Has(meta.ExpandRealizations) {
		dbCreditRealizations, err := entity.Edges.CreditAllocationsOrErr()
		if err != nil {
			return flatfee.Charge{}, fmt.Errorf("mapping flat fee charge [id=%s]: %w", entity.ID, err)
		}

		charge.State.CreditRealizations = lo.Map(dbCreditRealizations, func(entity *entdb.ChargeFlatFeeCreditAllocations, _ int) creditrealization.Realization {
			return creditrealization.MapFromDB(entity)
		})

		dbPaymentState, err := entity.Edges.PaymentOrErr()
		if _, ok := lo.ErrorsAs[*entdb.NotLoadedError](err); ok {
			return flatfee.Charge{}, fmt.Errorf("payment state not loaded for flat fee charge [id=%s]", entity.ID)
		}

		if dbPaymentState != nil {
			charge.State.Payment = lo.ToPtr(payment.MapInvoicedFromDB(dbPaymentState))
		}

		dbAccruedUsage, err := entity.Edges.InvoicedUsageOrErr()
		if _, ok := lo.ErrorsAs[*entdb.NotLoadedError](err); ok {
			return flatfee.Charge{}, fmt.Errorf("accrued usage not loaded for flat fee charge [id=%s]", entity.ID)
		}

		if dbAccruedUsage != nil {
			charge.State.AccruedUsage = lo.ToPtr(invoicedusage.MapAccruedUsageFromDB(dbAccruedUsage))
		}
	}

	return charge, nil
}

// proRatingConfigFromDB converts a DB ProRatingModeAdapterEnum to a ProRatingConfig.
func proRatingConfigFromDB(pr flatfee.ProRatingModeAdapterEnum) productcatalog.ProRatingConfig {
	switch pr {
	case flatfee.ProratePricesProratingAdapterMode:
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
func proRatingConfigToDB(pc productcatalog.ProRatingConfig) (flatfee.ProRatingModeAdapterEnum, error) {
	if !pc.Enabled {
		return flatfee.NoProratingAdapterMode, nil
	}

	if pc.Mode == productcatalog.ProRatingModeProratePrices {
		return flatfee.ProratePricesProratingAdapterMode, nil
	}

	return "", fmt.Errorf("invalid pro rating mode: %s", pc.Mode)
}
