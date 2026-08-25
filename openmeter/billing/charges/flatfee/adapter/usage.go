package adapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	dbchargeflatfeerun "github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfeerun"
	dbchargeflatfeeruninvoicedusage "github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfeeruninvoicedusage"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ flatfee.ChargeInvoicedUsageAdapter = (*adapter)(nil)

func (a *adapter) CreateInvoicedUsage(ctx context.Context, input flatfee.CreateInvoicedUsageInput) (invoicedusage.AccruedUsage, error) {
	if err := input.Validate(); err != nil {
		return invoicedusage.AccruedUsage{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (invoicedusage.AccruedUsage, error) {
		if _, err := tx.db.ChargeFlatFeeRun.UpdateOneID(input.RunID.ID).
			Where(dbchargeflatfeerun.Namespace(input.RunID.Namespace)).
			SetLineID(input.LineID).
			SetInvoiceID(input.InvoiceID).
			Save(ctx); err != nil {
			return invoicedusage.AccruedUsage{}, fmt.Errorf("updating flat fee run invoice refs [run_id=%s]: %w", input.RunID.ID, err)
		}

		create := tx.db.ChargeFlatFeeRunInvoicedUsage.Create().
			SetRunID(input.RunID.ID)

		create = invoicedusage.Create(create, input.RunID.Namespace, input.InvoicedUsage)

		entity, err := create.Save(ctx)
		if err != nil {
			return invoicedusage.AccruedUsage{}, err
		}

		return invoicedusage.MapAccruedUsageFromDB(entity), nil
	})
}

func (a *adapter) DeleteInvoicedUsage(ctx context.Context, id models.NamespacedID) error {
	if err := id.Validate(); err != nil {
		return err
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		if err := tx.db.ChargeFlatFeeRunInvoicedUsage.DeleteOneID(id.ID).
			Where(dbchargeflatfeeruninvoicedusage.NamespaceEQ(id.Namespace)).
			Exec(ctx); err != nil {
			return fmt.Errorf("deleting flat fee invoiced usage [id=%s]: %w", id.ID, err)
		}

		return nil
	})
}
