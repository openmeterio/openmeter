package service

import (
	"context"
	"slices"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/pkg/clock"
)

var activePromotionalCreditPurchaseStatuses = []creditpurchase.Status{
	creditpurchase.StatusCreated,
	creditpurchase.StatusActive,
}

func (s *service) onPromotionalCreditPurchase(ctx context.Context, charge creditpurchase.Charge) (creditpurchase.Charge, error) {
	// Prevent re-processing of the charge
	if !slices.Contains(activePromotionalCreditPurchaseStatuses, charge.Status) {
		return creditpurchase.Charge{}, creditpurchase.ErrCreditPurchaseChargeNotActive.WithAttrs(charge.ErrorAttributes())
	}

	ledgerTransactionGroupReference, err := s.handler.OnPromotionalCreditPurchase(ctx, charge)
	if err != nil {
		return creditpurchase.Charge{}, err
	}

	grantRealization, err := s.adapter.CreateCreditGrant(ctx, charge.GetChargeID(), creditpurchase.CreateCreditGrantInput{
		TransactionGroupID: ledgerTransactionGroupReference.TransactionGroupID,
		GrantedAt:          clock.Now(),
	})
	if err != nil {
		return creditpurchase.Charge{}, err
	}

	charge.Realizations.CreditGrantRealization = &grantRealization

	charge.Status = creditpurchase.StatusFinal

	updatedBase, err := s.adapter.UpdateCharge(ctx, charge.ChargeBase)
	if err != nil {
		return creditpurchase.Charge{}, err
	}

	charge.ChargeBase = updatedBase

	return charge, nil
}
