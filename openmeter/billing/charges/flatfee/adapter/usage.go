package adapter

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfeerun"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func (a *adapter) CreateInvoicedUsage(ctx context.Context, input flatfee.CreateInvoicedUsageInput) (invoicedusage.AccruedUsage, error) {
	if err := input.Validate(); err != nil {
		return invoicedusage.AccruedUsage{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (invoicedusage.AccruedUsage, error) {
		run, err := tx.currentRunByChargeID(ctx, input.ChargeID)
		if err != nil {
			return invoicedusage.AccruedUsage{}, err
		}

		if _, err := tx.updateCurrentRunTotals(ctx, run, input.InvoicedUsage.Totals); err != nil {
			return invoicedusage.AccruedUsage{}, err
		}

		if _, err := tx.db.ChargeFlatFeeRun.UpdateOneID(run.ID).
			Where(chargeflatfeerun.Namespace(input.ChargeID.Namespace)).
			SetLineID(input.LineID).
			SetInvoiceID(input.InvoiceID).
			Save(ctx); err != nil {
			return invoicedusage.AccruedUsage{}, err
		}

		create := tx.db.ChargeFlatFeeRunInvoicedUsage.Create().
			SetRunID(run.ID)

		create = invoicedusage.Create(create, input.ChargeID.Namespace, input.InvoicedUsage)

		entity, err := create.Save(ctx)
		if err != nil {
			return invoicedusage.AccruedUsage{}, err
		}

		return invoicedusage.MapAccruedUsageFromDB(entity), nil
	})
}
