package creditpurchase

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/pkg/clock"
)

func (s *service) onPromotionalCreditPurchase(ctx context.Context, charge charges.CreditPurchaseCharge) (charges.CreditPurchaseCharge, error) {
	ledgerTransactionGroupReference, err := s.creditPurchaseHandler.OnPromotionalCreditPurchase(ctx, charge)
	if err != nil {
		return charges.CreditPurchaseCharge{}, err
	}

	charge.State.CreditGrantRealization = &charges.TimedLedgerTransactionGroupReference{
		LedgerTransactionGroupReference: ledgerTransactionGroupReference,
		Time:                            clock.Now(),
	}

	charge.Status = charges.ChargeStatusFinal

	return s.adapter.UpdateCreditPurchaseCharge(ctx, charge)
}
