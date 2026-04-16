package adapter

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
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
