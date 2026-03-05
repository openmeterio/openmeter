package adapter

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	dbchargeexternalpaymentsettlement "github.com/openmeterio/openmeter/openmeter/ent/db/chargeexternalpaymentsettlement"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var _ charges.ExternalPaymentSettlementAdapter = (*adapter)(nil)

func (a *adapter) CreateExternalPaymentSettlement(ctx context.Context, input charges.ExternalPaymentSettlementCreateInput) (charges.ExternalPaymentSettlement, error) {
	if err := input.Validate(); err != nil {
		return charges.ExternalPaymentSettlement{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (charges.ExternalPaymentSettlement, error) {
		create := tx.db.ChargeExternalPaymentSettlement.Create().
			SetNamespace(input.Namespace).
			SetAnnotations(input.Annotations).
			SetServicePeriodFrom(input.ServicePeriod.From).
			SetServicePeriodTo(input.ServicePeriod.To).
			SetStatus(input.Status).
			SetAmount(input.Amount)

		if input.Authorized != nil {
			create = create.SetAuthorizedTransactionGroupID(input.Authorized.TransactionGroupID).
				SetAuthorizedAt(input.Authorized.Time)
		}
		if input.Settled != nil {
			create = create.SetSettledTransactionGroupID(input.Settled.TransactionGroupID).
				SetSettledAt(input.Settled.Time)
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
