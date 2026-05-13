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
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// MapFlatFeeChargeFromDB converts a DB Charge entity (with loaded FlatFee edge) to a FlatFeeCharge.
func MapChargeFlatFeeFromDB(entity *entdb.ChargeFlatFee, expands meta.Expands) (flatfee.Charge, error) {
	chargeBase := MapChargeBaseFromDB(entity)

	charge := flatfee.Charge{
		ChargeBase: chargeBase,
	}

	if expands.Has(meta.ExpandRealizations) {
		realizations, err := mapRealizationsFromDB(entity)
		if err != nil {
			return flatfee.Charge{}, fmt.Errorf("mapping flat fee charge [id=%s]: %w", entity.ID, err)
		}
		charge.Realizations = realizations
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

func mapRealizationsFromDB(entity *entdb.ChargeFlatFee) (flatfee.Realizations, error) {
	dbRuns, err := entity.Edges.RunsOrErr()
	if err != nil {
		return flatfee.Realizations{}, fmt.Errorf("runs not loaded for flat fee charge [id=%s]: %w", entity.ID, err)
	}

	var realizations flatfee.Realizations
	for _, dbRun := range dbRuns {
		run, err := mapRealizationRunFromDB(dbRun)
		if err != nil {
			return flatfee.Realizations{}, fmt.Errorf("mapping flat fee realization run [id=%s]: %w", dbRun.ID, err)
		}

		if entity.CurrentRealizationRunID != nil && dbRun.ID == *entity.CurrentRealizationRunID {
			realizations.CurrentRun = &run
			continue
		}

		realizations.PriorRuns = append(realizations.PriorRuns, run)
	}

	if entity.CurrentRealizationRunID != nil && realizations.CurrentRun == nil {
		return flatfee.Realizations{}, fmt.Errorf("current realization run [id=%s] not loaded for flat fee charge [id=%s]", *entity.CurrentRealizationRunID, entity.ID)
	}

	return realizations, nil
}

func mapRealizationRunBaseFromDB(dbRun *entdb.ChargeFlatFeeRun) flatfee.RealizationRunBase {
	return flatfee.RealizationRunBase{
		ID: flatfee.RealizationRunID{
			Namespace: dbRun.Namespace,
			ID:        dbRun.ID,
		},
		ManagedModel: entutils.MapTimeMixinFromDB(dbRun),

		LineID:                    dbRun.LineID,
		InvoiceID:                 dbRun.InvoiceID,
		Type:                      dbRun.Type,
		InitialType:               dbRun.InitialType,
		ServicePeriod:             timeutil.ClosedPeriod{From: dbRun.ServicePeriodFrom.UTC(), To: dbRun.ServicePeriodTo.UTC()},
		AmountAfterProration:      dbRun.AmountAfterProration,
		Totals:                    totals.FromDB(dbRun),
		NoFiatTransactionRequired: dbRun.NoFiatTransactionRequired,
		Immutable:                 dbRun.Immutable,
	}
}

func mapRealizationRunFromDB(dbRun *entdb.ChargeFlatFeeRun) (flatfee.RealizationRun, error) {
	run := flatfee.RealizationRun{
		RealizationRunBase: mapRealizationRunBaseFromDB(dbRun),
	}

	dbCreditsAllocated, err := dbRun.Edges.CreditAllocationsOrErr()
	if _, ok := lo.ErrorsAs[*entdb.NotLoadedError](err); ok {
		return flatfee.RealizationRun{}, fmt.Errorf("credits allocated not loaded for flat fee charge run [id=%s]", dbRun.ID)
	}

	for _, credit := range dbCreditsAllocated {
		run.CreditRealizations = append(run.CreditRealizations, creditrealization.MapFromDB(credit))
	}

	dbInvoiceUsage, err := dbRun.Edges.InvoicedUsageOrErr()
	if _, ok := lo.ErrorsAs[*entdb.NotLoadedError](err); ok {
		return flatfee.RealizationRun{}, fmt.Errorf("invoice usage not loaded for flat fee charge run [id=%s]", dbRun.ID)
	}

	if dbInvoiceUsage != nil {
		usage := invoicedusage.MapAccruedUsageFromDB(dbInvoiceUsage)
		run.AccruedUsage = &usage
	}

	dbPayment, err := dbRun.Edges.PaymentOrErr()
	if _, ok := lo.ErrorsAs[*entdb.NotLoadedError](err); ok {
		return flatfee.RealizationRun{}, fmt.Errorf("payment not loaded for flat fee charge run [id=%s]", dbRun.ID)
	}

	if dbPayment != nil {
		paymentState := payment.MapInvoicedFromDB(dbPayment)
		run.Payment = &paymentState
	}

	return run, nil
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
