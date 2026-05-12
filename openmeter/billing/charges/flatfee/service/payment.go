package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *service) PostInvoicePaymentAuthorized(ctx context.Context, charge flatfee.Charge, lineWithHeader billing.StandardLineWithInvoiceHeader) error {
	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		if charge.Realizations.CurrentRun == nil {
			return fmt.Errorf("current run is required")
		}

		if charge.Realizations.CurrentRun.Payment != nil {
			return payment.ErrPaymentAlreadyAuthorized.
				WithAttrs(charge.ErrorAttributes()).
				WithAttrs(charge.Realizations.CurrentRun.Payment.ErrorAttributes())
		}

		ledgerTransactionGroupReference, err := s.handler.OnPaymentAuthorized(ctx, charge)
		if err != nil {
			return err
		}

		newPaymentSettlement := payment.InvoicedCreate{
			Namespace: charge.Namespace,
			LineID:    lineWithHeader.Line.ID,
			InvoiceID: lineWithHeader.Invoice.ID,
			Base: payment.Base{
				ServicePeriod: charge.Intent.ServicePeriod,
				Amount:        lineWithHeader.Line.Totals.Total,
				Authorized: &ledgertransaction.TimedGroupReference{
					GroupReference: ledgertransaction.GroupReference{
						TransactionGroupID: ledgerTransactionGroupReference.TransactionGroupID,
					},
					Time: clock.Now(),
				},
				Status: payment.StatusAuthorized,
			},
		}

		paymentSettlement, err := s.adapter.CreatePayment(ctx, charge.GetChargeID(), newPaymentSettlement)
		if err != nil {
			return err
		}

		charge.Realizations.CurrentRun.Payment = &paymentSettlement

		return nil
	})
}

func (s *service) PostInvoicePaymentSettled(ctx context.Context, charge flatfee.Charge, lineWithHeader billing.StandardLineWithInvoiceHeader) error {
	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		if charge.Realizations.CurrentRun == nil {
			return fmt.Errorf("current run is required")
		}

		if charge.Realizations.CurrentRun.Payment == nil {
			return payment.ErrCannotSettleNotAuthorizedPayment.WithAttrs(charge.ErrorAttributes())
		}

		paymentSettlement := *charge.Realizations.CurrentRun.Payment

		if paymentSettlement.LineID != lineWithHeader.Line.ID {
			return fmt.Errorf("payment settlement line ID does not match the line ID: %s != %s", paymentSettlement.LineID, lineWithHeader.Line.ID)
		}

		if paymentSettlement.Status != payment.StatusAuthorized {
			return payment.ErrPaymentAlreadySettled.WithAttrs(charge.ErrorAttributes())
		}

		ledgerTransactionGroupReference, err := s.handler.OnPaymentSettled(ctx, charge)
		if err != nil {
			return err
		}

		paymentSettlement.Settled = &ledgertransaction.TimedGroupReference{
			GroupReference: ledgertransaction.GroupReference{
				TransactionGroupID: ledgerTransactionGroupReference.TransactionGroupID,
			},
			Time: clock.Now(),
		}

		paymentSettlement.Status = payment.StatusSettled

		paymentSettlement, err = s.adapter.UpdatePayment(ctx, paymentSettlement)
		if err != nil {
			return err
		}

		charge.Realizations.CurrentRun.Payment = &paymentSettlement
		charge.Status = flatfee.StatusFinal

		_, err = s.adapter.UpdateCharge(ctx, charge.ChargeBase)
		if err != nil {
			return err
		}

		return nil
	})
}
