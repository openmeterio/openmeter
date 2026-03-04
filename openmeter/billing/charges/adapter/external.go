package adapter

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	dbchargeexternalpaymentsettlement "github.com/openmeterio/openmeter/openmeter/ent/db/chargeexternalpaymentsettlement"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var _ charges.ExternalPaymentSettlementAdapter = (*adapter)(nil)

func (a *adapter) CreateExternalPaymentSettlement(ctx context.Context, chargeID charges.ChargeID, paymentSettlement charges.ExternalPaymentSettlement) (charges.ExternalPaymentSettlement, error) {
	if err := chargeID.Validate(); err != nil {
		return charges.ExternalPaymentSettlement{}, err
	}

	if err := paymentSettlement.Validate(); err != nil {
		return charges.ExternalPaymentSettlement{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (charges.ExternalPaymentSettlement, error) {
		create := tx.db.ChargeExternalPaymentSettlement.Create().
			SetNamespace(chargeID.Namespace).
			SetChargeID(chargeID.ID).
			SetAnnotations(paymentSettlement.Annotations).
			SetServicePeriodFrom(paymentSettlement.ServicePeriod.From).
			SetServicePeriodTo(paymentSettlement.ServicePeriod.To).
			SetStatus(paymentSettlement.Status).
			SetAmount(paymentSettlement.Amount)

		if paymentSettlement.Authorized != nil {
			create = create.SetAuthorizedTransactionGroupID(paymentSettlement.Authorized.TransactionGroupID).
				SetAuthorizedAt(paymentSettlement.Authorized.Time)
		}
		if paymentSettlement.Settled != nil {
			create = create.SetSettledTransactionGroupID(paymentSettlement.Settled.TransactionGroupID).
				SetSettledAt(paymentSettlement.Settled.Time)
		}

		entity, err := create.Save(ctx)
		if err != nil {
			return charges.ExternalPaymentSettlement{}, err
		}

		return mapExternalPaymentSettlementFromDB(entity), nil
	})
}

func (a *adapter) UpdateExternalPaymentSettlement(ctx context.Context, paymentSettlement charges.ExternalPaymentSettlement) (charges.ExternalPaymentSettlement, error) {
	if err := paymentSettlement.Validate(); err != nil {
		return charges.ExternalPaymentSettlement{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (charges.ExternalPaymentSettlement, error) {
		update := tx.db.ChargeExternalPaymentSettlement.UpdateOneID(paymentSettlement.ID).
			Where(dbchargeexternalpaymentsettlement.Namespace(paymentSettlement.Namespace)).
			SetNillableDeletedAt(paymentSettlement.DeletedAt).
			SetAnnotations(paymentSettlement.Annotations).
			SetServicePeriodFrom(paymentSettlement.ServicePeriod.From).
			SetServicePeriodTo(paymentSettlement.ServicePeriod.To).
			SetStatus(paymentSettlement.Status).
			SetAmount(paymentSettlement.Amount)

		if paymentSettlement.Authorized != nil {
			update = update.SetAuthorizedTransactionGroupID(paymentSettlement.Authorized.TransactionGroupID).
				SetAuthorizedAt(paymentSettlement.Authorized.Time)
		} else {
			update = update.ClearAuthorizedTransactionGroupID().
				ClearAuthorizedAt()
		}

		if paymentSettlement.Settled != nil {
			update = update.SetSettledTransactionGroupID(paymentSettlement.Settled.TransactionGroupID).
				SetSettledAt(paymentSettlement.Settled.Time)
		} else {
			update = update.ClearSettledTransactionGroupID().
				ClearSettledAt()
		}

		entity, err := update.Save(ctx)
		if err != nil {
			return charges.ExternalPaymentSettlement{}, err
		}

		return mapExternalPaymentSettlementFromDB(entity), nil
	})
}
