package adapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/ent/db/chargeflatfeerunpayment"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var _ flatfee.ChargePaymentAdapter = (*adapter)(nil)

func (a *adapter) CreatePayment(ctx context.Context, runID flatfee.RealizationRunID, paymentSettlement payment.InvoicedCreate) (payment.Invoiced, error) {
	if err := runID.Validate(); err != nil {
		return payment.Invoiced{}, err
	}

	if err := paymentSettlement.Validate(); err != nil {
		return payment.Invoiced{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (payment.Invoiced, error) {
		create := tx.db.ChargeFlatFeeRunPayment.Create().
			SetRunID(runID.ID)

		create = payment.CreateInvoiced(create, paymentSettlement)

		entity, err := create.Save(ctx)
		if err != nil {
			return payment.Invoiced{}, fmt.Errorf("creating flat fee run payment [run_id=%s]: %w", runID.ID, err)
		}

		return payment.MapInvoicedFromDB(entity), nil
	})
}

func (a *adapter) UpdatePayment(ctx context.Context, paymentSettlement payment.Invoiced) (payment.Invoiced, error) {
	if err := paymentSettlement.Validate(); err != nil {
		return payment.Invoiced{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (payment.Invoiced, error) {
		update := tx.db.ChargeFlatFeeRunPayment.UpdateOneID(paymentSettlement.ID).
			Where(chargeflatfeerunpayment.Namespace(paymentSettlement.Namespace))

		updated := payment.UpdateInvoiced(update, paymentSettlement)

		entity, err := updated.Save(ctx)
		if err != nil {
			return payment.Invoiced{}, err
		}

		return payment.MapInvoicedFromDB(entity), nil
	})
}
