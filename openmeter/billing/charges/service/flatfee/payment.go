package flatfee

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/pkg/clock"
)

func (s *service) PostPaymentAuthorized(ctx context.Context, charge charges.FlatFeeCharge, lineWithHeader billing.StandardLineWithInvoiceHeader) error {
	if charge.State.Payment != nil {
		return charges.ErrPaymentAlreadyAuthorized.
			WithAttrs(charge.ErrorAttributes()).
			WithAttrs(charge.State.Payment.ErrorAttributes())
	}

	ledgerTransactionGroupReference, err := s.flatFeeHandler.OnFlatFeePaymentAuthorized(ctx, charge)
	if err != nil {
		return err
	}

	newPaymentSettlement := charges.StandardInvoicePaymentSettlementCreateInput{
		Namespace: charge.Namespace,
		LineID:    lineWithHeader.Line.ID,
		PaymentSettlementBase: charges.PaymentSettlementBase{
			ServicePeriod: charge.Intent.ServicePeriod,
			Amount:        lineWithHeader.Line.Totals.Total,
			Authorized: &charges.TimedLedgerTransactionGroupReference{
				LedgerTransactionGroupReference: charges.LedgerTransactionGroupReference{
					TransactionGroupID: ledgerTransactionGroupReference.TransactionGroupID,
				},
				Time: clock.Now(),
			},
			Status: charges.PaymentSettlementStatusAuthorized,
		},
	}

	paymentSettlement, err := s.adapter.CreateStandardInvoicePaymentSettlement(ctx, newPaymentSettlement)
	if err != nil {
		return err
	}

	charge.State.Payment = &paymentSettlement
	_, err = s.adapter.UpdateFlatFeeCharge(ctx, charge)

	return err
}

func (s *service) PostPaymentSettled(ctx context.Context, charge charges.FlatFeeCharge, lineWithHeader billing.StandardLineWithInvoiceHeader) error {
	if charge.State.Payment == nil {
		return charges.ErrCannotSettleNotAuthorizedPayment.WithAttrs(charge.ErrorAttributes())
	}

	paymentSettlement := *charge.State.Payment

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

	charge.State.Payment = &paymentSettlement
	charge.Status = charges.ChargeStatusFinal

	_, err = s.adapter.UpdateFlatFeeCharge(ctx, charge)
	if err != nil {
		return err
	}

	return nil
}
