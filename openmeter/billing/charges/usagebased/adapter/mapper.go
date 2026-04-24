package adapter

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/chargemeta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func MapChargeFromDB(entity *entdb.ChargeUsageBased, expands meta.Expands) (usagebased.Charge, error) {
	chargeBase := MapChargeBaseFromDB(entity)

	var realizations usagebased.RealizationRuns
	if expands.Has(meta.ExpandRealizations) {
		var err error

		realizations, err = MapRealizationRunsFromDB(entity)
		if err != nil {
			return usagebased.Charge{}, fmt.Errorf("mapping usage based charge [id=%s]: %w", entity.ID, err)
		}
	}

	return usagebased.Charge{
		ChargeBase:   chargeBase,
		Realizations: realizations,
	}, nil
}

func MapChargeBaseFromDB(entity *entdb.ChargeUsageBased) usagebased.ChargeBase {
	chargeMeta := chargemeta.MapFromDB(entity)

	return usagebased.ChargeBase{
		ManagedResource: chargeMeta.ManagedResource,
		Status:          entity.StatusDetailed,
		Intent: usagebased.Intent{
			Intent:         chargeMeta.Intent,
			InvoiceAt:      entity.InvoiceAt.UTC(),
			SettlementMode: entity.SettlementMode,
			FeatureKey:     entity.FeatureKey,
			Discounts:      lo.FromPtr(entity.Discounts),
			Price:          lo.FromPtr(entity.Price),
		},
		State: usagebased.State{
			CurrentRealizationRunID: entity.CurrentRealizationRunID,
			AdvanceAfter:            entity.AdvanceAfter,
			FeatureID:               entity.FeatureID,
		},
	}
}

// MapRealizationRunsFromDB converts a DB Charge entity (with loaded UsageBased edge) to a UsageBasedCharge.
func MapRealizationRunsFromDB(entity *entdb.ChargeUsageBased) (usagebased.RealizationRuns, error) {
	dbRuns, err := entity.Edges.RunsOrErr()
	if err != nil {
		return nil, fmt.Errorf("mapping usage based charge [id=%s]: %w", entity.ID, err)
	}

	runs, err := slicesx.MapWithErr(dbRuns, func(run *entdb.ChargeUsageBasedRuns) (usagebased.RealizationRun, error) {
		return MapRealizationRunFromDB(run)
	})
	if err != nil {
		return nil, fmt.Errorf("mapping usage based charge [id=%s]: %w", entity.ID, err)
	}

	if len(runs) == 0 {
		// Force nil value for easier testing
		runs = nil
	}

	// Let's keep the runs sorted by period end.
	slices.SortStableFunc(runs, func(a, b usagebased.RealizationRun) int {
		return cmp.Compare(a.ServicePeriodTo.UnixNano(), b.ServicePeriodTo.UnixNano())
	})

	return runs, nil
}

func MapRealizationRunBaseFromDB(dbRun *entdb.ChargeUsageBasedRuns) usagebased.RealizationRunBase {
	return usagebased.RealizationRunBase{
		ID: usagebased.RealizationRunID{
			Namespace: dbRun.Namespace,
			ID:        dbRun.ID,
		},
		ManagedModel: entutils.MapTimeMixinFromDB(dbRun),

		FeatureID:       dbRun.FeatureID,
		LineID:          dbRun.LineID,
		Type:            dbRun.Type,
		StoredAtLT:      dbRun.StoredAtLt.UTC(),
		ServicePeriodTo: dbRun.ServicePeriodTo.UTC(),
		MeteredQuantity: dbRun.MeteredQuantity,
		Totals:          totals.FromDB(dbRun),
	}
}

func MapRealizationRunFromDB(dbRun *entdb.ChargeUsageBasedRuns) (usagebased.RealizationRun, error) {
	run := usagebased.RealizationRun{
		RealizationRunBase: MapRealizationRunBaseFromDB(dbRun),
	}

	dbCreditsAllocated, err := dbRun.Edges.CreditAllocationsOrErr()
	if _, ok := lo.ErrorsAs[*entdb.NotLoadedError](err); ok {
		return usagebased.RealizationRun{}, fmt.Errorf("credits allocated not loaded for usage based charge run [id=%s]", dbRun.ID)
	}

	run.CreditsAllocated = lo.Map(dbCreditsAllocated, func(credit *entdb.ChargeUsageBasedRunCreditAllocations, _ int) creditrealization.Realization {
		return creditrealization.MapFromDB(credit)
	})

	dbInvoiceUsage, err := dbRun.Edges.InvoicedUsageOrErr()
	if _, ok := lo.ErrorsAs[*entdb.NotLoadedError](err); ok {
		return usagebased.RealizationRun{}, fmt.Errorf("invoice usage not loaded for usage based charge run [id=%s]", dbRun.ID)
	}

	if dbInvoiceUsage != nil {
		run.InvoiceUsage = lo.ToPtr(invoicedusage.MapAccruedUsageFromDB(dbInvoiceUsage))
	}

	dbPayment, err := dbRun.Edges.PaymentOrErr()
	if _, ok := lo.ErrorsAs[*entdb.NotLoadedError](err); ok {
		return usagebased.RealizationRun{}, fmt.Errorf("payment not loaded for usage based charge run [id=%s]", dbRun.ID)
	}

	if dbPayment != nil {
		run.Payment = lo.ToPtr(payment.MapInvoicedFromDB(dbPayment))
	}

	return run, nil
}
