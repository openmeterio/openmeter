package service

import (
	"context"
	"slices"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/pkg/clock"
)

var activePromotionalCreditPurchaseStatuses = []meta.ChargeStatus{
	meta.ChargeStatusCreated,
	meta.ChargeStatusActive,
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

	charge.State.CreditGrantRealization = &ledgertransaction.TimedGroupReference{
		GroupReference: ledgerTransactionGroupReference,
		Time:           clock.Now(),
	}

	charge.Status = meta.ChargeStatusFinal

	charge, err = s.adapter.UpdateCharge(ctx, charge)
	if err != nil {
		return creditpurchase.Charge{}, err
	}

	return charge, nil
}
