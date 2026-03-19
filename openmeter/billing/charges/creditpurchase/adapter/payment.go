package adapter

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/ent/db/chargecreditpurchaseexternalpayment"
	"github.com/openmeterio/openmeter/openmeter/ent/db/chargecreditpurchaseinvoicedpayment"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func (a *adapter) CreateExternalPayment(ctx context.Context, chargeID meta.ChargeID, in payment.ExternalCreateInput) (payment.External, error) {
	if err := chargeID.Validate(); err != nil {
		return payment.External{}, err
	}

	if err := in.Validate(); err != nil {
		return payment.External{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (payment.External, error) {
		create := tx.db.ChargeCreditPurchaseExternalPayment.Create().
			SetChargeID(chargeID.ID)

		create = payment.CreateExternal(create, in)

		entity, err := create.Save(ctx)
		if err != nil {
			return payment.External{}, err
		}

		return payment.MapExternalFromDB(entity), nil
	})
}

func (a *adapter) UpdateExternalPayment(ctx context.Context, in payment.External) (payment.External, error) {
	if err := in.Validate(); err != nil {
		return payment.External{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (payment.External, error) {
		update := tx.db.ChargeCreditPurchaseExternalPayment.UpdateOneID(in.ID).
			Where(chargecreditpurchaseexternalpayment.Namespace(in.Namespace))

		updated := payment.UpdateExternal(update, in)

		entity, err := updated.Save(ctx)
		if err != nil {
			return payment.External{}, err
		}

		return payment.MapExternalFromDB(entity), nil
	})
}

func (a *adapter) CreateInvoicedPayment(ctx context.Context, chargeID meta.ChargeID, in payment.InvoicedCreate) (payment.Invoiced, error) {
	if err := chargeID.Validate(); err != nil {
		return payment.Invoiced{}, err
	}

	if err := in.Validate(); err != nil {
		return payment.Invoiced{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (payment.Invoiced, error) {
		create := tx.db.ChargeCreditPurchaseInvoicedPayment.Create().
			SetChargeID(chargeID.ID)

		create = payment.CreateInvoiced(create, in)

		entity, err := create.Save(ctx)
		if err != nil {
			return payment.Invoiced{}, err
		}

		return payment.MapInvoicedFromDB(entity), nil
	})
}

func (a *adapter) UpdateInvoicedPayment(ctx context.Context, in payment.Invoiced) (payment.Invoiced, error) {
	if err := in.Validate(); err != nil {
		return payment.Invoiced{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (payment.Invoiced, error) {
		update := tx.db.ChargeCreditPurchaseInvoicedPayment.UpdateOneID(in.ID).
			Where(chargecreditpurchaseinvoicedpayment.Namespace(in.Namespace))

		updated := payment.UpdateInvoiced(update, in)

		entity, err := updated.Save(ctx)
		if err != nil {
			return payment.Invoiced{}, err
		}

		return payment.MapInvoicedFromDB(entity), nil
	})
}
