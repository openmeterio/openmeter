package service

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/pkg/clock"
)

func (s *service) onPromotionalCreditPurchase(ctx context.Context, charge creditpurchase.Charge) (creditpurchase.Charge, error) {
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
