package run

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/pkg/clock"
)

type BookInvoicedPaymentAuthorizedInput struct {
	Charge  usagebased.Charge
	Run     usagebased.RealizationRun
	Invoice billing.StandardInvoice
	Line    billing.StandardLine
}

func (i BookInvoicedPaymentAuthorizedInput) Validate() error {
	if err := i.Charge.Validate(); err != nil {
		return fmt.Errorf("charge: %w", err)
	}

	if err := i.Run.Validate(); err != nil {
		return fmt.Errorf("run: %w", err)
	}

	if err := i.Line.Validate(); err != nil {
		return fmt.Errorf("line: %w", err)
	}

	if i.Invoice.ID == "" {
		return fmt.Errorf("invoice id is required")
	}

	if i.Run.LineID == nil {
		return fmt.Errorf("run %s has no linked standard line", i.Run.ID.ID)
	}

	if *i.Run.LineID != i.Line.ID {
		return fmt.Errorf("run %s already linked to a different line", i.Run.ID.ID)
	}

	if i.Run.Payment != nil {
		return payment.ErrPaymentAlreadyAuthorized.WithAttrs(i.Charge.ErrorAttributes())
	}

	return nil
}

type BookInvoicedPaymentAuthorizedResult struct {
	Run     usagebased.RealizationRun
	Payment *payment.Invoiced
}

func (s *Service) BookInvoicedPaymentAuthorized(ctx context.Context, in BookInvoicedPaymentAuthorizedInput) (BookInvoicedPaymentAuthorizedResult, error) {
	if err := in.Validate(); err != nil {
		return BookInvoicedPaymentAuthorizedResult{}, err
	}

	input := usagebased.OnPaymentAuthorizedInput{
		Charge: in.Charge,
		Run:    in.Run,
	}
	if err := input.Validate(); err != nil {
		return BookInvoicedPaymentAuthorizedResult{}, fmt.Errorf("validate on payment authorized input: %w", err)
	}

	ledgerTransactionRef, err := s.handler.OnPaymentAuthorized(ctx, input)
	if err != nil {
		return BookInvoicedPaymentAuthorizedResult{}, fmt.Errorf("on usage-based payment authorized: %w", err)
	}

	paymentRealization, err := s.adapter.CreateRunPayment(ctx, in.Run.ID, payment.InvoicedCreate{
		Namespace: in.Charge.Namespace,
		LineID:    in.Line.ID,
		InvoiceID: in.Invoice.ID,
		Base: payment.Base{
			ServicePeriod: in.Line.Period,
			Amount:        in.Line.Totals.Total,
			Authorized: &ledgertransaction.TimedGroupReference{
				GroupReference: ledgertransaction.GroupReference{
					TransactionGroupID: ledgerTransactionRef.TransactionGroupID,
				},
				Time: clock.Now(),
			},
			Status: payment.StatusAuthorized,
		},
	})
	if err != nil {
		return BookInvoicedPaymentAuthorizedResult{}, fmt.Errorf("create invoiced payment for run %s: %w", in.Run.ID.ID, err)
	}

	in.Run.Payment = &paymentRealization

	return BookInvoicedPaymentAuthorizedResult{
		Run:     in.Run,
		Payment: &paymentRealization,
	}, nil
}

type SettleInvoicedPaymentInput struct {
	Charge  usagebased.Charge
	Run     usagebased.RealizationRun
	Invoice billing.StandardInvoice
	Line    billing.StandardLine
}

func (i SettleInvoicedPaymentInput) Validate() error {
	if err := i.Charge.Validate(); err != nil {
		return fmt.Errorf("charge: %w", err)
	}

	if err := i.Run.Validate(); err != nil {
		return fmt.Errorf("run: %w", err)
	}

	if err := i.Line.Validate(); err != nil {
		return fmt.Errorf("line: %w", err)
	}

	if i.Invoice.ID == "" {
		return fmt.Errorf("invoice id is required")
	}

	if i.Run.LineID == nil {
		return fmt.Errorf("run %s has no linked standard line", i.Run.ID.ID)
	}

	if *i.Run.LineID != i.Line.ID {
		return fmt.Errorf("run %s already linked to a different line", i.Run.ID.ID)
	}

	if i.Run.Payment == nil {
		return payment.ErrCannotSettleNotAuthorizedPayment.WithAttrs(i.Charge.ErrorAttributes())
	}

	if i.Run.Payment.LineID != i.Line.ID {
		return fmt.Errorf("payment line ID does not match the line ID: %s != %s", i.Run.Payment.LineID, i.Line.ID)
	}

	if i.Run.Payment.Status != payment.StatusAuthorized {
		return payment.ErrPaymentAlreadySettled.WithAttrs(i.Charge.ErrorAttributes())
	}

	return nil
}

type SettleInvoicedPaymentResult struct {
	Run     usagebased.RealizationRun
	Payment *payment.Invoiced
}

func (s *Service) SettleInvoicedPayment(ctx context.Context, in SettleInvoicedPaymentInput) (SettleInvoicedPaymentResult, error) {
	if err := in.Validate(); err != nil {
		return SettleInvoicedPaymentResult{}, err
	}

	input := usagebased.OnPaymentSettledInput{
		Charge: in.Charge,
		Run:    in.Run,
	}
	if err := input.Validate(); err != nil {
		return SettleInvoicedPaymentResult{}, fmt.Errorf("validate on payment settled input: %w", err)
	}

	ledgerTransactionRef, err := s.handler.OnPaymentSettled(ctx, input)
	if err != nil {
		return SettleInvoicedPaymentResult{}, fmt.Errorf("on usage-based payment settled: %w", err)
	}

	paymentRealization := *in.Run.Payment
	paymentRealization.Settled = &ledgertransaction.TimedGroupReference{
		GroupReference: ledgertransaction.GroupReference{
			TransactionGroupID: ledgerTransactionRef.TransactionGroupID,
		},
		Time: clock.Now(),
	}
	paymentRealization.Status = payment.StatusSettled

	paymentRealization, err = s.adapter.UpdateRunPayment(ctx, paymentRealization)
	if err != nil {
		return SettleInvoicedPaymentResult{}, fmt.Errorf("update invoiced payment for run %s: %w", in.Run.ID.ID, err)
	}

	in.Run.Payment = &paymentRealization

	return SettleInvoicedPaymentResult{
		Run:     in.Run,
		Payment: &paymentRealization,
	}, nil
}
