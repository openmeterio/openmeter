package flatfee

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
)

func (s *service) PostPaymentAuthorized(ctx context.Context, charge charges.FlatFeeCharge, lineWithHeader billing.StandardLineWithInvoiceHeader) error {
	ledgerTransactionGroupReference, err := s.flatFeeHandler.OnFlatFeePaymentAuthorized(ctx, charge)
	if err != nil {
		return err
	}

	charge.State.AuthorizedTransaction = &ledgerTransactionGroupReference

	_, err = s.adapter.UpdateFlatFeeCharge(ctx, charge)
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
