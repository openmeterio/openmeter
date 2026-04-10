package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *service) PostInvoiceDraftCreated(ctx context.Context, charge creditpurchase.Charge, lineWithHeader billing.StandardLineWithInvoiceHeader) error {
	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		ledgerTransactionGroupReference, err := s.handler.OnCreditPurchaseInitiated(ctx, charge)
		if err != nil {
			return err
		}

		if _, err := s.adapter.CreateCreditGrant(ctx, charge.GetChargeID(), creditpurchase.CreateCreditGrantInput{
			TransactionGroupID: ledgerTransactionGroupReference.TransactionGroupID,
			GrantedAt:          clock.Now(),
		}); err != nil {
			return err
		}

		if ledgerTransactionGroupReference.TransactionGroupID != "" {
			if err := s.lineage.BackfillAdvanceLineageSegments(ctx, lineage.BackfillAdvanceLineageSegmentsInput{
				Namespace:                 charge.Namespace,
				CustomerID:                charge.Intent.CustomerID,
				Currency:                  charge.Intent.Currency,
				Amount:                    charge.Intent.CreditAmount,
				BackingTransactionGroupID: ledgerTransactionGroupReference.TransactionGroupID,
			}); err != nil {
				return err
			}
		}

		charge.Status = creditpurchase.StatusActive

		_, err = s.adapter.UpdateCharge(ctx, charge.ChargeBase)
		return err
	})
}

// PostInvoicePaymentAuthorized is called when the invoice is approved/issued.
// It's invoked from the billing service's PostUpdate hook, already within a transaction.
func (s *service) PostInvoicePaymentAuthorized(ctx context.Context, charge creditpurchase.Charge, lineWithHeader billing.StandardLineWithInvoiceHeader) error {
	if charge.Realizations.InvoiceSettlement != nil {
		return fmt.Errorf("invoice settlement already already authorized - settlement already exists: %s", charge.Realizations.InvoiceSettlement.InvoiceID)
	}

	ledgerTransactionGroupReference, err := s.handler.OnCreditPurchasePaymentAuthorized(ctx, charge)
	if err != nil {
		return err
	}

	newPaymentSettlement := payment.InvoicedCreate{
		Namespace: charge.Namespace,
		Base: payment.Base{
			ServicePeriod: charge.Intent.ServicePeriod,
			Amount:        charge.Intent.CreditAmount,
			Authorized: &ledgertransaction.TimedGroupReference{
				GroupReference: ledgerTransactionGroupReference,
				Time:           clock.Now(),
			},
			Status: payment.StatusAuthorized,
		},
		InvoiceID: lineWithHeader.Invoice.ID,
		LineID:    lineWithHeader.Line.ID,
	}

	_, err = s.adapter.CreateInvoicedPayment(ctx, charge.GetChargeID(), newPaymentSettlement)
	if err != nil {
		return err
	}

	return nil
}

// PostInvoicePaymentSettled is called when the invoice is paid.
// It's invoked from the billing service's PostUpdate hook, already within a transaction.
func (s *service) PostInvoicePaymentSettled(ctx context.Context, charge creditpurchase.Charge, lineWithHeader billing.StandardLineWithInvoiceHeader) error {
	// Idempotency check: if already settled, skip processing
	if charge.Realizations.InvoiceSettlement == nil {
		return fmt.Errorf("invoice settlement not found")
	}

	if charge.Realizations.InvoiceSettlement.Settled != nil {
		return fmt.Errorf("invoice settlement already settled")
	}

	paymentSettlement := *charge.Realizations.InvoiceSettlement

	ledgerTransactionGroupReference, err := s.handler.OnCreditPurchasePaymentSettled(ctx, charge)
	if err != nil {
		return err
	}

	paymentSettlement.Settled = &ledgertransaction.TimedGroupReference{
		GroupReference: ledgerTransactionGroupReference,
		Time:           clock.Now(),
	}

	paymentSettlement.Status = payment.StatusSettled

	if _, err := s.adapter.UpdateInvoicedPayment(ctx, paymentSettlement); err != nil {
		return err
	}

	// Update charge status to final
	charge.Status = creditpurchase.StatusFinal

	if _, err := s.adapter.UpdateCharge(ctx, charge.ChargeBase); err != nil {
		return err
	}

	return nil
}
