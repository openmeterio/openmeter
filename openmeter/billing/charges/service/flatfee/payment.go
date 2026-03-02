package flatfee

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/pkg/clock"
)

func (s *service) PostPaymentAuthorized(ctx context.Context, charge charges.FlatFeeCharge, lineWithHeader billing.StandardLineWithInvoiceHeader) error {
	if charge.State.Payment.Authorized != nil {
		return errors.New("payment already authorized")
	}

	ledgerTransactionGroupReference, err := s.flatFeeHandler.OnFlatFeePaymentAuthorized(ctx, charge)
	if err != nil {
		return err
	}

	paymentState := charge.State.Payment
	if paymentState == nil {
		paymentState = &charges.StandardInvoicePaymentSettlement{}
	}

	paymentState.Authorized = &charges.TimedLedgerTransactionGroupReference{
		LedgerTransactionGroupReference: charges.LedgerTransactionGroupReference{
			TransactionGroupID: ledgerTransactionGroupReference.TransactionGroupID,
		},
		Time: clock.Now(),
	}

	_, err = s.adapter.UpsertStandardInvoicePaymentSettlement(ctx, charge.ID, paymentState)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) PostPaymentSettled(ctx context.Context, charge charges.FlatFeeCharge, lineWithHeader billing.StandardLineWithInvoiceHeader) error {
	ledgerTransactionGroupReference, err := s.flatFeeHandler.OnFlatFeePaymentSettled(ctx, charge)
	if err != nil {
		return err
	}

	charge.State.SettledTransaction = &ledgerTransactionGroupReference
	charge.Status = charges.ChargeStatusFinal

	_, err = s.adapter.UpdateFlatFeeCharge(ctx, charge)
	if err != nil {
		return err
	}

	return nil
}
