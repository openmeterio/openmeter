package adapter

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfeepayment"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func (a *adapter) CreatePayment(ctx context.Context, chargeID meta.ChargeID, paymentSettlement payment.InvoicedCreate) (payment.Invoiced, error) {
	if err := paymentSettlement.Validate(); err != nil {
		return payment.Invoiced{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (payment.Invoiced, error) {
		create := tx.db.ChargeFlatFeePayment.Create().
			SetChargeID(chargeID.ID)

		create = payment.CreateInvoiced(create, paymentSettlement)

		entity, err := create.Save(ctx)
		if err != nil {
			return payment.Invoiced{}, err
		}

		return payment.MapInvoicedFromDB(entity), nil
	})
}

func (a *adapter) UpdatePayment(ctx context.Context, paymentSettlement payment.Invoiced) (payment.Invoiced, error) {
	if err := paymentSettlement.Validate(); err != nil {
		return payment.Invoiced{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (payment.Invoiced, error) {
		update := tx.db.ChargeFlatFeePayment.UpdateOneID(paymentSettlement.ID).
			Where(chargeflatfeepayment.Namespace(paymentSettlement.Namespace))

		updated := payment.UpdateInvoiced(update, paymentSettlement)

		entity, err := updated.Save(ctx)
		if err != nil {
			return payment.Invoiced{}, err
		}

		return payment.MapInvoicedFromDB(entity), nil
	})
}
