package adapter

import (
	"fmt"
	"slices"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/chargemeta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

// FromDB converts a DB ChargeFlatFee entity to a flat fee charge.
func FromDB(entity *entdb.ChargeFlatFee, expands meta.Expands) (flatfee.Charge, error) {
	mappedMeta, err := chargemeta.FromDB(entity, entity.Edges)
	if err != nil {
		return flatfee.Charge{}, fmt.Errorf("mapping flat fee charge meta [id=%s]: %w", entity.ID, err)
	}

	return fromDBWithMeta(entity, mappedMeta, expands)
}

func FromDBWithCurrency(entity *entdb.ChargeFlatFee, currency currencies.Currency, expands meta.Expands) (flatfee.Charge, error) {
	mappedMeta, err := chargemeta.FromDBWithCurrency(entity, currency)
	if err != nil {
		return flatfee.Charge{}, fmt.Errorf("mapping flat fee charge meta [id=%s]: %w", entity.ID, err)
	}

	return fromDBWithMeta(entity, mappedMeta, expands)
}

func fromDBWithMeta(entity *entdb.ChargeFlatFee, mappedMeta meta.Charge, expands meta.Expands) (flatfee.Charge, error) {
	base, err := fromDBBase(entity, mappedMeta)
	if err != nil {
		return flatfee.Charge{}, fmt.Errorf("mapping flat fee charge base [id=%s]: %w", entity.ID, err)
	}

	charge := flatfee.Charge{
		ChargeBase: base,
	}

	if expands.Has(meta.ExpandRealizations) {
		realizations, err := fromDBRealizations(entity)
		if err != nil {
			return flatfee.Charge{}, fmt.Errorf("mapping flat fee charge [id=%s]: %w", entity.ID, err)
		}
		charge.Realizations = realizations
	}

	return charge, nil
}

func fromDBRealizations(entity *entdb.ChargeFlatFee) (flatfee.Realizations, error) {
	dbRuns, err := entity.Edges.RunsOrErr()
	if err != nil {
		return flatfee.Realizations{}, fmt.Errorf("runs not loaded for flat fee charge [id=%s]: %w", entity.ID, err)
	}

	var realizations flatfee.Realizations
	for _, dbRun := range dbRuns {
		run, err := fromDBRun(dbRun)
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

func fromDBRunBase(dbRun *entdb.ChargeFlatFeeRun) flatfee.RealizationRunBase {
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

func fromDBRun(dbRun *entdb.ChargeFlatFeeRun) (flatfee.RealizationRun, error) {
	run := flatfee.RealizationRun{
		RealizationRunBase: fromDBRunBase(dbRun),
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

func sortDetailedLines(lines flatfee.DetailedLines) {
	slices.SortStableFunc(lines, stddetailedline.Compare[flatfee.DetailedLine])
}

func fromDBBaseWithCurrency(entity *entdb.ChargeFlatFee, currency currencies.Currency) (flatfee.ChargeBase, error) {
	mappedMeta, err := chargemeta.FromDBWithCurrency(entity, currency)
	if err != nil {
		return flatfee.ChargeBase{}, fmt.Errorf("mapping charge meta: %w", err)
	}

	return fromDBBase(entity, mappedMeta)
}

func fromDBBase(entity *entdb.ChargeFlatFee, mappedMeta meta.Charge) (flatfee.ChargeBase, error) {
	var percentageDiscounts *billing.PercentageDiscount
	if entity.Discounts != nil {
		percentageDiscounts = entity.Discounts.Percentage
	}

	var costBasisIntent *costbasis.Intent
	var resolvedCostBasis *costbasis.State
	var costBasisID *string
	if entity.CostBasisID != nil {
		if entity.Edges.CostBasis == nil {
			return flatfee.ChargeBase{}, fmt.Errorf("cost basis not loaded for flat fee charge [id=%s,cost_basis_id=%s]", entity.ID, *entity.CostBasisID)
		}

		if entity.Edges.CostBasis.ID != *entity.CostBasisID {
			return flatfee.ChargeBase{}, fmt.Errorf("cost basis ID mismatch for flat fee charge [id=%s,cost_basis_id=%s,edge_id=%s]", entity.ID, *entity.CostBasisID, entity.Edges.CostBasis.ID)
		}

		mappedCostBasis, err := costbasis.Get(entity.Edges.CostBasis)
		if err != nil {
			return flatfee.ChargeBase{}, fmt.Errorf("mapping cost basis: %w", err)
		}

		costBasisID = lo.ToPtr(*entity.CostBasisID)
		costBasisIntent = &mappedCostBasis.Intent
		resolvedCostBasis = mappedCostBasis.State
	} else if entity.Edges.CostBasis != nil {
		return flatfee.ChargeBase{}, fmt.Errorf("cost basis edge loaded without a reference for flat fee charge [id=%s,edge_id=%s]", entity.ID, entity.Edges.CostBasis.ID)
	}

	return flatfee.ChargeBase{
		ManagedResource: mappedMeta.ManagedResource,
		Status:          entity.StatusDetailed,
		State: flatfee.State{
			AdvanceAfter:         mappedMeta.AdvanceAfter,
			FeatureID:            entity.FeatureID,
			AmountAfterProration: entity.AmountAfterProration,
			CostBasisID:          costBasisID,
			ResolvedCostBasis:    resolvedCostBasis,
		},
		Intent: flatfee.NewOverridableIntent(flatfee.Intent{
			Intent:         mappedMeta.Intent,
			SettlementMode: entity.SettlementMode,
			FeatureKey:     entity.FeatureKey,
			CostBasis:      costBasisIntent,
			IntentMutableFields: flatfee.IntentMutableFields{
				IntentMutableFields:   mappedMeta.IntentMutableFields,
				InvoiceAt:             entity.InvoiceAt.UTC(),
				IntentDeletedAt:       convert.TimePtrIn(entity.IntentDeletedAt, time.UTC),
				PaymentTerm:           entity.PaymentTerm,
				PercentageDiscounts:   percentageDiscounts,
				ProRating:             fromDBProRatingConfig(entity.ProRating),
				AmountBeforeProration: entity.AmountBeforeProration,
			},
		}, fromDBOverride(entity.Edges.IntentOverride)),
	}, nil
}

// fromDBProRatingConfig converts a DB ProRatingModeAdapterEnum to a ProRatingConfig.
func fromDBProRatingConfig(pr flatfee.ProRatingModeAdapterEnum) productcatalog.ProRatingConfig {
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
