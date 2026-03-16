package adapter

import (
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

// MapUsageBasedChargeFromDB converts a DB Charge entity (with loaded UsageBased edge) to a UsageBasedCharge.
func MapUsageBasedChargeFromDB(entity *entdb.ChargeUsageBased, chargeMeta meta.Charge, expands meta.Expands) (usagebased.Charge, error) {
	charge := usagebased.Charge{
		ManagedResource: chargeMeta.ManagedResource,
		Status:          chargeMeta.Status,
		Intent: usagebased.Intent{
			Intent:         chargeMeta.Intent,
			InvoiceAt:      entity.InvoiceAt.UTC(),
			SettlementMode: entity.SettlementMode,
			FeatureKey:     entity.FeatureKey,
			Discounts:      lo.FromPtr(entity.Discounts),
			Price:          lo.FromPtr(entity.Price),
		},
	}

	if expands.Has(meta.ExpandRealizations) {
		dbRuns, err := entity.Edges.RunsOrErr()
		if err != nil {
			return usagebased.Charge{}, fmt.Errorf("mapping usage based charge [id=%s]: %w", entity.ID, err)
		}

		runs, err := slicesx.MapWithErr(dbRuns, func(run *entdb.ChargeUsageBasedRuns) (usagebased.RealizationRun, error) {
			return MapRealizationRunFromDB(run)
		})
		if err != nil {
			return usagebased.Charge{}, fmt.Errorf("mapping usage based charge [id=%s]: %w", entity.ID, err)
		}

		if len(runs) > 0 {
			// Force nil value for easier testing
			charge.State.RealizationRuns = runs
		}
	}

	return charge, nil
}

func MapRealizationRunFromDB(dbRun *entdb.ChargeUsageBasedRuns) (usagebased.RealizationRun, error) {
	run := usagebased.RealizationRun{
		NamespacedID: models.NamespacedID{
			Namespace: dbRun.Namespace,
			ID:        dbRun.ID,
		},
		ManagedModel: entutils.MapTimeMixinFromDB(dbRun),
		Type:         dbRun.Type,
		AsOf:         dbRun.Asof,
		MeterValue:   dbRun.MeterValue,
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
