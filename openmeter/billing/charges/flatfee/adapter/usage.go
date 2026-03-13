package adapter

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func (a *adapter) CreateInvoicedUsage(ctx context.Context, chargeID meta.ChargeID, invoicedUsage invoicedusage.AccruedUsage) (invoicedusage.AccruedUsage, error) {
	if err := invoicedUsage.Validate(); err != nil {
		return invoicedusage.AccruedUsage{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (invoicedusage.AccruedUsage, error) {
		create := tx.db.ChargeFlatFeeInvoicedUsage.Create().
			SetChargeID(chargeID.ID)

		create = invoicedusage.Create(create, chargeID.Namespace, invoicedUsage)

		entity, err := create.Save(ctx)
		if err != nil {
			return invoicedusage.AccruedUsage{}, err
		}

		return invoicedusage.MapAccruedUsageFromDB(entity), nil
	})
}
