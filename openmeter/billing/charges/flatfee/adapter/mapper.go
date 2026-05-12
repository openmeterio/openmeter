package adapter

import (
	"fmt"
	"slices"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/chargemeta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/samber/mo"
)

// MapFlatFeeChargeFromDB converts a DB Charge entity (with loaded FlatFee edge) to a FlatFeeCharge.
func MapChargeFlatFeeFromDB(entity *entdb.ChargeFlatFee, expands meta.Expands) (flatfee.Charge, error) {
	chargeBase := MapChargeBaseFromDB(entity)

	charge := flatfee.Charge{
		ChargeBase: chargeBase,
	}

	if expands.Has(meta.ExpandRealizations) {
		dbRun, err := entity.Edges.CurrentRunOrErr()
		if err != nil {
			if _, ok := lo.ErrorsAs[*entdb.NotFoundError](err); !ok {
				return flatfee.Charge{}, fmt.Errorf("mapping flat fee charge [id=%s]: %w", entity.ID, err)
			}
		}

		if dbRun != nil {
			if err := mapRealizationsFromCurrentRun(dbRun, &charge); err != nil {
				return flatfee.Charge{}, fmt.Errorf("mapping flat fee realization run [charge_id=%s]: %w", entity.ID, err)
			}
		}
	}

	return charge, nil
}

func mapRunDetailedLineFromDB(dbLine *entdb.ChargeFlatFeeRunDetailedLine) (flatfee.DetailedLine, error) {
	line := stddetailedline.FromDB(
		dbLine,
		stddetailedline.BackfillTaxConfig(
			lo.EmptyableToPtr(dbLine.TaxConfig),
			dbLine.TaxBehavior,
			taxCodeIDFromEnt(dbLine.Edges.TaxCode),
		),
	)

	return line, line.Validate()
}

func mapRealizationsFromCurrentRun(dbRun *entdb.ChargeFlatFeeRun, charge *flatfee.Charge) error {
	dbCreditsAllocated, err := dbRun.Edges.CreditAllocationsOrErr()
	if _, ok := lo.ErrorsAs[*entdb.NotLoadedError](err); ok {
		return fmt.Errorf("credits allocated not loaded for flat fee charge run [id=%s]", dbRun.ID)
	}

	for _, credit := range dbCreditsAllocated {
		charge.Realizations.CreditRealizations = append(charge.Realizations.CreditRealizations, creditrealization.MapFromDB(credit))
	}

	dbInvoiceUsage, err := dbRun.Edges.InvoicedUsageOrErr()
	if _, ok := lo.ErrorsAs[*entdb.NotLoadedError](err); ok {
		return fmt.Errorf("invoice usage not loaded for flat fee charge run [id=%s]", dbRun.ID)
	}

	if dbInvoiceUsage != nil {
		usage := invoicedusage.MapAccruedUsageFromDB(dbInvoiceUsage)
		charge.Realizations.AccruedUsage = &usage
	}

	dbPayment, err := dbRun.Edges.PaymentOrErr()
	if _, ok := lo.ErrorsAs[*entdb.NotLoadedError](err); ok {
		return fmt.Errorf("payment not loaded for flat fee charge run [id=%s]", dbRun.ID)
	}

	if dbPayment != nil {
		paymentState := payment.MapInvoicedFromDB(dbPayment)
		charge.Realizations.Payment = &paymentState
	}

	dbDetailedLines, err := dbRun.Edges.DetailedLinesOrErr()
	if err == nil {
		lines := make(flatfee.DetailedLines, 0, len(dbDetailedLines))
		for _, dbLine := range dbDetailedLines {
			line, err := mapRunDetailedLineFromDB(dbLine)
			if err != nil {
				return err
			}

			lines = append(lines, line)
		}

		sortDetailedLines(lines)
		charge.Realizations.DetailedLines = mo.Some(lines)
	}

	return nil
}

func taxCodeIDFromEnt(resolvedTaxCode *entdb.TaxCode) *string {
	if resolvedTaxCode == nil {
		return nil
	}

	return lo.ToPtr(resolvedTaxCode.ID)
}

func sortDetailedLines(lines flatfee.DetailedLines) {
	slices.SortStableFunc(lines, stddetailedline.Compare[flatfee.DetailedLine])
}

func MapChargeBaseFromDB(entity *entdb.ChargeFlatFee) flatfee.ChargeBase {
	var percentageDiscounts *productcatalog.PercentageDiscount
	if entity.Discounts != nil {
		percentageDiscounts = entity.Discounts.Percentage
	}

	mappedMeta := chargemeta.MapFromDB(entity)

	return flatfee.ChargeBase{
		ManagedResource: mappedMeta.ManagedResource,
		Status:          entity.StatusDetailed,
		State: flatfee.State{
			AdvanceAfter:         mappedMeta.AdvanceAfter,
			FeatureID:            entity.FeatureID,
			AmountAfterProration: entity.AmountAfterProration,
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
		},
	}
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
