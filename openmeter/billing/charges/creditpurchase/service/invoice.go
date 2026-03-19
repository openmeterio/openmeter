package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/pkg/clock"
)

func (s *service) onInvoiceSettlementPurchase(ctx context.Context, charge creditpurchase.Charge) (creditpurchase.Charge, error) {
	ledgerTransactionGroupReference, err := s.handler.OnCreditPurchaseInitiated(ctx, charge)
	if err != nil {
		return creditpurchase.Charge{}, err
	}

	charge.State.CreditGrantRealization = &ledgertransaction.TimedGroupReference{
		GroupReference: ledgerTransactionGroupReference,
		Time:           clock.Now(),
	}
	charge.Status = meta.ChargeStatusActive

	return s.adapter.UpdateCharge(ctx, charge)
}

// PostLineAssignedToInvoice creates the initial InvoiceSettlement (payment.Invoiced) record
// when a credit purchase gathering line is assigned to a standard invoice.
func (s *service) PostLineAssignedToInvoice(ctx context.Context, input creditpurchase.PostLineAssignedToInvoiceInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validating input: %w", err)
	}

	charge := input.Charge

	// Idempotency: if InvoiceSettlement is already set, skip
	if charge.State.InvoiceSettlement != nil {
		return nil
	}

	settlement, err := charge.Intent.Settlement.AsInvoiceSettlement()
	if err != nil {
		return fmt.Errorf("getting invoice settlement from intent: %w", err)
	}

	totalCost := charge.Intent.CreditAmount.Mul(settlement.CostBasis)

	invoicedPayment, err := s.adapter.CreateInvoicedPayment(ctx, charge.GetChargeID(), payment.InvoicedCreate{
		Base: payment.Base{
			ServicePeriod: charge.Intent.ServicePeriod,
			Status:        payment.StatusAuthorized,
			Amount:        totalCost,
			Authorized:    charge.State.CreditGrantRealization,
		},
		Namespace: charge.Namespace,
		LineID:    input.LineID,
		InvoiceID: input.InvoiceID,
	})
	if err != nil {
		return fmt.Errorf("creating invoiced payment: %w", err)
	}

	charge.State.InvoiceSettlement = &invoicedPayment

	if _, err = s.adapter.UpdateCharge(ctx, charge); err != nil {
		return fmt.Errorf("updating charge with invoice settlement: %w", err)
	}

	return nil
}

// HandleInvoicePaymentAuthorized is called when the invoice is approved/issued.
// It's invoked from the billing service's PostUpdate hook, already within a transaction.
func (s *service) HandleInvoicePaymentAuthorized(ctx context.Context, charge creditpurchase.Charge, lineWithHeader billing.StandardLineWithInvoiceHeader) error {
	// Idempotency check: if already authorized via invoice approval (not just initial credit grant), skip processing.
	// LinkInvoicedPayment sets Authorized to CreditGrantRealization, so we check if it's been updated
	// by this handler (which would have a different transaction group ID).
	if charge.State.InvoiceSettlement == nil {
		return nil
	}

	if charge.State.InvoiceSettlement.Authorized != nil &&
		charge.State.CreditGrantRealization != nil &&
		charge.State.InvoiceSettlement.Authorized.TransactionGroupID != charge.State.CreditGrantRealization.GroupReference.TransactionGroupID {
		return nil
	}

	ledgerTransactionGroupReference, err := s.handler.OnCreditPurchasePaymentAuthorized(ctx, charge)
	if err != nil {
		return err
	}

	paymentSettlement := *charge.State.InvoiceSettlement
	paymentSettlement.Authorized = &ledgertransaction.TimedGroupReference{
		GroupReference: ledgerTransactionGroupReference,
		Time:           clock.Now(),
	}

	paymentSettlement, err = s.adapter.UpdateInvoicedPayment(ctx, paymentSettlement)
	if err != nil {
		return err
	}

	charge.State.InvoiceSettlement = &paymentSettlement

	_, err = s.adapter.UpdateCharge(ctx, charge)
	return err
}

// HandleInvoicePaymentSettled is called when the invoice is paid.
// It's invoked from the billing service's PostUpdate hook, already within a transaction.
func (s *service) HandleInvoicePaymentSettled(ctx context.Context, charge creditpurchase.Charge, lineWithHeader billing.StandardLineWithInvoiceHeader) error {
	// Idempotency check: if already settled, skip processing
	if charge.State.InvoiceSettlement == nil {
		return nil
	}

	if charge.State.InvoiceSettlement.Settled != nil {
		return nil
	}

	paymentSettlement := *charge.State.InvoiceSettlement

	ledgerTransactionGroupReference, err := s.handler.OnCreditPurchasePaymentSettled(ctx, charge)
	if err != nil {
		return err
	}

	paymentSettlement.Settled = &ledgertransaction.TimedGroupReference{
		GroupReference: ledgerTransactionGroupReference,
		Time:           clock.Now(),
	}

	paymentSettlement.Status = payment.StatusSettled

	updatedPayment, err := s.adapter.UpdateInvoicedPayment(ctx, paymentSettlement)
	if err != nil {
		return err
	}

	charge.State.InvoiceSettlement = &updatedPayment

	// Update charge status to final
	charge.Status = meta.ChargeStatusFinal

	if _, err := s.adapter.UpdateCharge(ctx, charge); err != nil {
		return err
	}

	return nil
}
