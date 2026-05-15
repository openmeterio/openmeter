package service

import (
	"context"
	"slices"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
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

	if ledgerTransactionGroupReference.TransactionGroupID != "" {
		if err := s.lineage.BackfillAdvanceLineageSegments(ctx, lineage.BackfillAdvanceLineageSegmentsInput{
			Namespace:                 charge.Namespace,
			CustomerID:                charge.Intent.CustomerID,
			Currency:                  charge.Intent.Currency,
			Amount:                    charge.Intent.CreditAmount,
			BackingTransactionGroupID: ledgerTransactionGroupReference.TransactionGroupID,
		}); err != nil {
			return creditpurchase.Charge{}, err
		}
	}

	charge.Status = creditpurchase.StatusFinal

	updatedBase, err := s.adapter.UpdateCharge(ctx, charge.ChargeBase)
	if err != nil {
		return creditpurchase.Charge{}, err
	}

	charge.ChargeBase = updatedBase

	return charge, nil
}
