package adapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	dbchargeusagebasedruninvoicedusage "github.com/openmeterio/openmeter/openmeter/ent/db/chargeusagebasedruninvoicedusage"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ usagebased.RealizationRunInvoiceUsageAdapter = (*adapter)(nil)

func (a *adapter) CreateRunInvoicedUsage(ctx context.Context, runID usagebased.RealizationRunID, usage invoicedusage.AccruedUsage) (invoicedusage.AccruedUsage, error) {
	if err := runID.Validate(); err != nil {
		return invoicedusage.AccruedUsage{}, err
	}

	if err := usage.Validate(); err != nil {
		return invoicedusage.AccruedUsage{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (invoicedusage.AccruedUsage, error) {
		create := tx.db.ChargeUsageBasedRunInvoicedUsage.Create().
			SetRunID(runID.ID)

		create = invoicedusage.Create(create, runID.Namespace, usage)

		entity, err := create.Save(ctx)
		if err != nil {
			return invoicedusage.AccruedUsage{}, err
		}

		return invoicedusage.MapAccruedUsageFromDB(entity), nil
	})
}

func (a *adapter) DeleteRunInvoicedUsage(ctx context.Context, id models.NamespacedID) error {
	if err := id.Validate(); err != nil {
		return err
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		if err := tx.db.ChargeUsageBasedRunInvoicedUsage.DeleteOneID(id.ID).
			Where(dbchargeusagebasedruninvoicedusage.NamespaceEQ(id.Namespace)).
			Exec(ctx); err != nil {
			return fmt.Errorf("deleting usage-based invoiced usage [id=%s]: %w", id.ID, err)
		}

		return nil
	})
}
