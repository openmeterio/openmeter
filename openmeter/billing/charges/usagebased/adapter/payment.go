package adapter

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/ent/db/chargeusagebasedrunpayment"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var _ usagebased.RealizationRunPaymentAdapter = (*adapter)(nil)

func (a *adapter) CreateRunPayment(ctx context.Context, runID usagebased.RealizationRunID, in payment.InvoicedCreate) (payment.Invoiced, error) {
	if err := runID.Validate(); err != nil {
		return payment.Invoiced{}, err
	}

	if err := in.Validate(); err != nil {
		return payment.Invoiced{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (payment.Invoiced, error) {
		create := tx.db.ChargeUsageBasedRunPayment.Create().
			SetRunID(runID.ID)

		create = payment.CreateInvoiced(create, in)

		entity, err := create.Save(ctx)
		if err != nil {
			return payment.Invoiced{}, err
		}

		return payment.MapInvoicedFromDB(entity), nil
	})
}

func (a *adapter) UpdateRunPayment(ctx context.Context, in payment.Invoiced) (payment.Invoiced, error) {
	if err := in.Validate(); err != nil {
		return payment.Invoiced{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (payment.Invoiced, error) {
		update := tx.db.ChargeUsageBasedRunPayment.UpdateOneID(in.ID).
			Where(chargeusagebasedrunpayment.Namespace(in.Namespace))

		update = payment.UpdateInvoiced(update, in)

		entity, err := update.Save(ctx)
		if err != nil {
			return payment.Invoiced{}, err
		}

		return payment.MapInvoicedFromDB(entity), nil
	})
}
