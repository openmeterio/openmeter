package flatfee

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *service) PostPaymentAuthorized(ctx context.Context, charge charges.FlatFeeCharge, lineWithHeader billing.StandardLineWithInvoiceHeader) error {
	paymentState, stateExists := s.getPaymentStateOrEmpty(charge, lineWithHeader.Line)

	if paymentState.Authorized != nil {
		return errors.New("payment already authorized")
	}

	ledgerTransactionGroupReference, err := s.flatFeeHandler.OnFlatFeePaymentAuthorized(ctx, charge)
	if err != nil {
		return err
	}

	paymentState.Authorized = &charges.TimedLedgerTransactionGroupReference{
		LedgerTransactionGroupReference: charges.LedgerTransactionGroupReference{
			TransactionGroupID: ledgerTransactionGroupReference.TransactionGroupID,
		},
		Time: clock.Now(),
	}

	paymentState.Status = charges.StandardInvoicePaymentSettlementStatusAuthorized

	if !stateExists {
		_, err = s.adapter.CreateStandardInvoicePaymentSettlement(ctx, charge.GetChargeID(), paymentState)
	} else {
		_, err = s.adapter.UpdateStandardInvoicePaymentSettlement(ctx, paymentState)
	}

	return err
}

func (s *service) PostPaymentSettled(ctx context.Context, charge charges.FlatFeeCharge, lineWithHeader billing.StandardLineWithInvoiceHeader) error {
	paymentState, stateExists := s.getPaymentStateOrEmpty(charge, lineWithHeader.Line)

	if !stateExists {
		return errors.New("payment state does not exist: it should have been already authorized")
	}

	if paymentState.Authorized == nil {
		return errors.New("payment is not authorized: it should have been already authorized")
	}

	if paymentState.Settled != nil {
		return errors.New("payment already settled")
	}

	ledgerTransactionGroupReference, err := s.flatFeeHandler.OnFlatFeePaymentSettled(ctx, charge)
	if err != nil {
		return err
	}

	paymentState.Settled = &charges.TimedLedgerTransactionGroupReference{
		LedgerTransactionGroupReference: charges.LedgerTransactionGroupReference{
			TransactionGroupID: ledgerTransactionGroupReference.TransactionGroupID,
		},
		Time: clock.Now(),
	}

	paymentState.Status = charges.StandardInvoicePaymentSettlementStatusSettled

	_, err = s.adapter.UpdateStandardInvoicePaymentSettlement(ctx, paymentState)
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

func (s *service) getPaymentStateOrEmpty(charge charges.FlatFeeCharge, line *billing.StandardLine) (charges.StandardInvoicePaymentSettlement, bool) {
	if charge.State.Payment != nil {
		return *charge.State.Payment, true
	}

	return charges.StandardInvoicePaymentSettlement{
		NamespacedID: models.NamespacedID{
			Namespace: charge.Namespace,
		},
		LineID:        line.ID,
		ServicePeriod: charge.Intent.ServicePeriod,
		Amount:        line.Totals.Total,
	}, false
}
