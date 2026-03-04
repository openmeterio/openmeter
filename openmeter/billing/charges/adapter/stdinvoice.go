package adapter

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/ent/db/chargestandardinvoicepaymentsettlement"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var _ charges.StandardInvoiceRealizationAdapter = (*adapter)(nil)

func (a *adapter) CreateStandardInvoicePaymentSettlement(ctx context.Context, chargeID charges.ChargeID, paymentState charges.StandardInvoicePaymentSettlement) (charges.StandardInvoicePaymentSettlement, error) {
	if err := chargeID.Validate(); err != nil {
		return charges.StandardInvoicePaymentSettlement{}, err
	}

	if err := paymentState.Validate(); err != nil {
		return charges.StandardInvoicePaymentSettlement{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (charges.StandardInvoicePaymentSettlement, error) {
		create := tx.db.ChargeStandardInvoicePaymentSettlement.Create().
			SetNamespace(chargeID.Namespace).
			SetChargeID(chargeID.ID).
			SetAnnotations(paymentState.Annotations).
			SetNillableDeletedAt(paymentState.DeletedAt).
			SetLineID(paymentState.LineID).
			SetServicePeriodFrom(paymentState.ServicePeriod.From).
			SetServicePeriodTo(paymentState.ServicePeriod.To).
			SetStatus(paymentState.Status).
			SetAmount(paymentState.Amount)

		if paymentState.Authorized != nil {
			create = create.SetAuthorizedTransactionGroupID(paymentState.Authorized.TransactionGroupID).
				SetAuthorizedAt(paymentState.Authorized.Time)
		}
		if paymentState.Settled != nil {
			create = create.SetSettledTransactionGroupID(paymentState.Settled.TransactionGroupID).
				SetSettledAt(paymentState.Settled.Time)
		}

		entity, err := create.Save(ctx)
		if err != nil {
			return charges.StandardInvoicePaymentSettlement{}, err
		}

		return mapStandardInvoicePaymentSettlementFromDB(entity), nil
	})
}

func (a *adapter) UpdateStandardInvoicePaymentSettlement(ctx context.Context, paymentState charges.StandardInvoicePaymentSettlement) (charges.StandardInvoicePaymentSettlement, error) {
	if err := paymentState.Validate(); err != nil {
		return charges.StandardInvoicePaymentSettlement{}, err
	}

	if err := paymentState.NamespacedID.Validate(); err != nil {
		return charges.StandardInvoicePaymentSettlement{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (charges.StandardInvoicePaymentSettlement, error) {
		update := tx.db.ChargeStandardInvoicePaymentSettlement.UpdateOneID(paymentState.NamespacedID.ID).
			Where(chargestandardinvoicepaymentsettlement.Namespace(paymentState.NamespacedID.Namespace)).
			SetAnnotations(paymentState.Annotations).
			SetNillableDeletedAt(paymentState.DeletedAt).
			SetServicePeriodFrom(paymentState.ServicePeriod.From).
			SetServicePeriodTo(paymentState.ServicePeriod.To).
			SetStatus(paymentState.Status).
			SetAmount(paymentState.Amount)

		if paymentState.Authorized != nil {
			update = update.SetAuthorizedTransactionGroupID(paymentState.Authorized.TransactionGroupID).
				SetAuthorizedAt(paymentState.Authorized.Time)
		} else {
			update = update.ClearAuthorizedTransactionGroupID().
				ClearAuthorizedAt()
		}

		if paymentState.Settled != nil {
			update = update.SetSettledTransactionGroupID(paymentState.Settled.TransactionGroupID).
				SetSettledAt(paymentState.Settled.Time)
		} else {
			update = update.ClearSettledTransactionGroupID().
				ClearSettledAt()
		}

		entity, err := update.Save(ctx)
		if err != nil {
			return charges.StandardInvoicePaymentSettlement{}, err
		}

		return mapStandardInvoicePaymentSettlementFromDB(entity), nil
	})
}

func (a *adapter) CreateStandardInvoiceAccruedUsage(ctx context.Context, chargeID charges.ChargeID, accruedUsage charges.StandardInvoiceAccruedUsage) (charges.StandardInvoiceAccruedUsage, error) {
	if err := chargeID.Validate(); err != nil {
		return charges.StandardInvoiceAccruedUsage{}, err
	}

	if err := accruedUsage.Validate(); err != nil {
		return charges.StandardInvoiceAccruedUsage{}, err
	}

	var trnsGroupID *string
	if accruedUsage.LedgerTransaction != nil {
		trnsGroupID = &accruedUsage.LedgerTransaction.TransactionGroupID
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (charges.StandardInvoiceAccruedUsage, error) {
		entity, err := tx.db.ChargeStandardInvoiceAccruedUsage.Create().
			SetNamespace(chargeID.Namespace).
			SetChargeID(chargeID.ID).
			SetAnnotations(accruedUsage.Annotations).
			SetNillableDeletedAt(accruedUsage.DeletedAt).
			SetNillableLineID(accruedUsage.LineID).
			SetServicePeriodFrom(accruedUsage.ServicePeriod.From).
			SetServicePeriodTo(accruedUsage.ServicePeriod.To).
			SetMutable(accruedUsage.Mutable).
			SetNillableLedgerTransactionGroupID(trnsGroupID).
			// Totals
			SetAmount(accruedUsage.Totals.Amount).
			SetChargesTotal(accruedUsage.Totals.ChargesTotal).
			SetDiscountsTotal(accruedUsage.Totals.DiscountsTotal).
			SetTaxesInclusiveTotal(accruedUsage.Totals.TaxesInclusiveTotal).
			SetTaxesExclusiveTotal(accruedUsage.Totals.TaxesExclusiveTotal).
			SetTaxesTotal(accruedUsage.Totals.TaxesTotal).
			SetCreditsTotal(accruedUsage.Totals.CreditsTotal).
			SetTotal(accruedUsage.Totals.Total).
			Save(ctx)
		if err != nil {
			return charges.StandardInvoiceAccruedUsage{}, err
		}

		return mapStandardInvoiceAccruedUsageFromDB(entity), nil
	})
}
