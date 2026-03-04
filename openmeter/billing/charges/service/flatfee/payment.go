package flatfee

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *service) PostPaymentAuthorized(ctx context.Context, charge charges.FlatFeeCharge, lineWithHeader billing.StandardLineWithInvoiceHeader) error {
	paymentSettlement, paymentSettlementExists := s.getPaymentSettlementOrEmpty(charge, lineWithHeader.Line)

	if paymentSettlementExists {
		return charges.ErrPaymentAlreadyAuthorized.
			WithAttrs(charge.ErrorAttributes()).
			WithAttrs(paymentSettlement.ErrorAttributes())
	}

	ledgerTransactionGroupReference, err := s.flatFeeHandler.OnFlatFeePaymentAuthorized(ctx, charge)
	if err != nil {
		return err
	}

	paymentSettlement.Authorized = &charges.TimedLedgerTransactionGroupReference{
		LedgerTransactionGroupReference: charges.LedgerTransactionGroupReference{
			TransactionGroupID: ledgerTransactionGroupReference.TransactionGroupID,
		},
		Time: clock.Now(),
	}

	paymentSettlement.Status = charges.PaymentSettlementStatusAuthorized

	_, err = s.adapter.CreateStandardInvoicePaymentSettlement(ctx, charge.GetChargeID(), paymentSettlement)

	return err
}

func (s *service) PostPaymentSettled(ctx context.Context, charge charges.FlatFeeCharge, lineWithHeader billing.StandardLineWithInvoiceHeader) error {
	paymentSettlement, paymentSettlementExists := s.getPaymentSettlementOrEmpty(charge, lineWithHeader.Line)

	if !paymentSettlementExists {
		return charges.ErrCannotSettleNotAuthorizedPayment.WithAttrs(charge.ErrorAttributes())
	}

	if paymentSettlement.Status != charges.PaymentSettlementStatusAuthorized {
		return charges.ErrPaymentAlreadySettled.WithAttrs(charge.ErrorAttributes())
	}

	ledgerTransactionGroupReference, err := s.flatFeeHandler.OnFlatFeePaymentSettled(ctx, charge)
	if err != nil {
		return err
	}

	paymentSettlement.Settled = &charges.TimedLedgerTransactionGroupReference{
		LedgerTransactionGroupReference: charges.LedgerTransactionGroupReference{
			TransactionGroupID: ledgerTransactionGroupReference.TransactionGroupID,
		},
		Time: clock.Now(),
	}

	paymentSettlement.Status = charges.PaymentSettlementStatusSettled

	paymentSettlement, err = s.adapter.UpdateStandardInvoicePaymentSettlement(ctx, paymentSettlement)
	if err != nil {
		return err
	}

	charge.Status = charges.ChargeStatusFinal

	_, err = s.adapter.UpdateFlatFeeCharge(ctx, charge)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) getPaymentSettlementOrEmpty(charge charges.FlatFeeCharge, line *billing.StandardLine) (charges.StandardInvoicePaymentSettlement, bool) {
	if charge.State.Payment != nil {
		return *charge.State.Payment, true
	}

	return charges.StandardInvoicePaymentSettlement{
		PaymentSettlementBase: charges.PaymentSettlementBase{
			NamespacedID: models.NamespacedID{
				Namespace: charge.Namespace,
			},
			ServicePeriod: charge.Intent.ServicePeriod,
			Amount:        line.Totals.Total,
		},
		LineID: line.ID,
	}, false
}
