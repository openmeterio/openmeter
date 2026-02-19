package adapter

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/openmeter/charges"
	"github.com/openmeterio/openmeter/openmeter/ent/db/chargestandardinvoicerealization"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func (a *adapter) CreateStandardInvoiceRealization(ctx context.Context, chargeID charges.ChargeID, realization charges.StandardInvoiceRealization) (charges.StandardInvoiceRealization, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (charges.StandardInvoiceRealization, error) {
		entity, err := tx.db.ChargeStandardInvoiceRealization.Create().
			SetNamespace(chargeID.Namespace).
			SetChargeID(chargeID.ID).
			SetLineID(realization.LineID).
			SetServicePeriodFrom(realization.ServicePeriod.From.In(time.UTC)).
			SetServicePeriodTo(realization.ServicePeriod.To.In(time.UTC)).
			SetStatus(realization.Status).
			SetMeteredServicePeriodQuantity(realization.MeteredServicePeriodQuantity).
			SetMeteredPreServicePeriodQuantity(realization.MeteredPreServicePeriodQuantity).
			SetAmount(realization.Totals.Amount).
			SetTaxesTotal(realization.Totals.TaxesTotal).
			SetTaxesInclusiveTotal(realization.Totals.TaxesInclusiveTotal).
			SetTaxesExclusiveTotal(realization.Totals.TaxesExclusiveTotal).
			SetChargesTotal(realization.Totals.ChargesTotal).
			SetDiscountsTotal(realization.Totals.DiscountsTotal).
			SetTotal(realization.Totals.Total).
			Save(ctx)
		if err != nil {
			return charges.StandardInvoiceRealization{}, err
		}

		return mapStandardInvoiceRealizationFromDB(entity), nil
	})
}

func (a *adapter) UpdateStandardInvoiceRealizationByID(ctx context.Context, chargeID charges.ChargeID, realization charges.StandardInvoiceRealization) (charges.StandardInvoiceRealization, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (charges.StandardInvoiceRealization, error) {
		entity, err := tx.db.ChargeStandardInvoiceRealization.UpdateOneID(realization.ID).
			Where(chargestandardinvoicerealization.ChargeID(chargeID.ID)).
			Where(chargestandardinvoicerealization.Namespace(chargeID.Namespace)).
			SetServicePeriodFrom(realization.ServicePeriod.From.In(time.UTC)).
			SetServicePeriodTo(realization.ServicePeriod.To.In(time.UTC)).
			SetStatus(realization.Status).
			SetMeteredServicePeriodQuantity(realization.MeteredServicePeriodQuantity).
			SetMeteredPreServicePeriodQuantity(realization.MeteredPreServicePeriodQuantity).
			SetAmount(realization.Totals.Amount).
			SetTaxesTotal(realization.Totals.TaxesTotal).
			SetTaxesInclusiveTotal(realization.Totals.TaxesInclusiveTotal).
			SetTaxesExclusiveTotal(realization.Totals.TaxesExclusiveTotal).
			SetChargesTotal(realization.Totals.ChargesTotal).
			SetDiscountsTotal(realization.Totals.DiscountsTotal).
			SetTotal(realization.Totals.Total).
			Save(ctx)
		if err != nil {
			return charges.StandardInvoiceRealization{}, err
		}
		return mapStandardInvoiceRealizationFromDB(entity), nil
	})
}
