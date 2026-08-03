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

func (s *service) postInvoicePaymentAuthorized(ctx context.Context, charge flatfee.Charge, lineWithHeader billing.StandardLineWithInvoiceHeader) error {
	if err := lineWithHeader.Validate(); err != nil {
		return fmt.Errorf("validating line with header: %w", err)
	}

	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		run, err := charge.Realizations.GetByLineID(lineWithHeader.Line.ID)
		if err != nil {
			return err
		}

		if err := validatePaymentRunForLine(charge, run, lineWithHeader); err != nil {
			return err
		}

		fiatAmount := lineWithHeader.Line.Totals.Total
		if run.NoFiatTransactionRequired || fiatAmount.IsZero() {
			return nil
		}

		if run.Payment != nil {
			return payment.ErrPaymentAlreadyAuthorized.
				WithAttrs(charge.ErrorAttributes()).
				WithAttrs(run.Payment.ErrorAttributes())
		}

		eventAt := clock.Now()
		ledgerTransactionGroupReference, err := s.handler.OnPaymentAuthorized(ctx, flatfee.OnPaymentAuthorizedInput{
			Charge:     charge,
			Run:        run,
			EventAt:    eventAt,
			FiatAmount: fiatAmount,
		})
		if err != nil {
			return err
		}

		newPaymentSettlement := payment.InvoicedCreate{
			Namespace: charge.Namespace,
			LineID:    lineWithHeader.Line.ID,
			InvoiceID: lineWithHeader.Invoice.ID,
			Base: payment.Base{
				ServicePeriod: run.ServicePeriod,
				FiatAmount:    fiatAmount,
				Authorized: &ledgertransaction.TimedGroupReference{
					GroupReference: ledgertransaction.GroupReference{
						TransactionGroupID: ledgerTransactionGroupReference.TransactionGroupID,
					},
					Time: eventAt,
				},
				Status: payment.StatusAuthorized,
			},
		}

		if _, err := s.adapter.CreatePayment(ctx, run.ID, newPaymentSettlement); err != nil {
			return err
		}

		return nil
	})
}

func (s *service) postInvoicePaymentSettled(ctx context.Context, charge flatfee.Charge, lineWithHeader billing.StandardLineWithInvoiceHeader) error {
	if err := lineWithHeader.Validate(); err != nil {
		return fmt.Errorf("validating line with header: %w", err)
	}

	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		run, err := charge.Realizations.GetByLineID(lineWithHeader.Line.ID)
		if err != nil {
			return err
		}

		if err := validatePaymentRunForLine(charge, run, lineWithHeader); err != nil {
			return err
		}

		fiatAmount := lineWithHeader.Line.Totals.Total
		if run.NoFiatTransactionRequired || fiatAmount.IsZero() {
			return nil
		}

		if run.Payment == nil {
			return payment.ErrCannotSettleNotAuthorizedPayment.WithAttrs(charge.ErrorAttributes())
		}

		paymentSettlement := *run.Payment

		if paymentSettlement.LineID != lineWithHeader.Line.ID {
			return fmt.Errorf("payment settlement line ID does not match the line ID: %s != %s", paymentSettlement.LineID, lineWithHeader.Line.ID)
		}

		if paymentSettlement.Status != payment.StatusAuthorized {
			return payment.ErrPaymentAlreadySettled.WithAttrs(charge.ErrorAttributes())
		}

		eventAt := clock.Now()
		ledgerTransactionGroupReference, err := s.handler.OnPaymentSettled(ctx, flatfee.OnPaymentSettledInput{
			Charge:     charge,
			Run:        run,
			EventAt:    eventAt,
			FiatAmount: fiatAmount,
		})
		if err != nil {
			return err
		}

		paymentSettlement.Settled = &ledgertransaction.TimedGroupReference{
			GroupReference: ledgertransaction.GroupReference{
				TransactionGroupID: ledgerTransactionGroupReference.TransactionGroupID,
			},
			Time: eventAt,
		}

		paymentSettlement.Status = payment.StatusSettled

		paymentSettlement, err = s.adapter.UpdatePayment(ctx, paymentSettlement)
		if err != nil {
			return err
		}

		return nil
	})
}

func validatePaymentRunForLine(charge flatfee.Charge, run flatfee.RealizationRun, lineWithHeader billing.StandardLineWithInvoiceHeader) error {
	if lineWithHeader.Line.ChargeID == nil || *lineWithHeader.Line.ChargeID != charge.ID {
		return fmt.Errorf("line charge id must match charge")
	}

	if run.LineID == nil || *run.LineID != lineWithHeader.Line.ID {
		return fmt.Errorf("realization run line id must match standard line")
	}

	if run.InvoiceID == nil || *run.InvoiceID != lineWithHeader.Invoice.ID {
		return fmt.Errorf("realization run invoice id must match invoice")
	}

	return nil
}
