package creditpurchase

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/pkg/clock"
)

func (s *service) PostCreate(ctx context.Context, charge charges.CreditPurchaseCharge) (charges.CreditPurchaseCharge, error) {
	switch charge.Intent.Settlement.Type() {
	case charges.CreditPurchaseSettlementTypePromotional:
		return s.onPromotionalCreditPurchase(ctx, charge)
	case charges.CreditPurchaseSettlementTypeInvoice:
		return charges.CreditPurchaseCharge{}, charges.ErrUnsupported
	case charges.CreditPurchaseSettlementTypeExternal:
		return charges.CreditPurchaseCharge{}, charges.ErrUnsupported
	default:
		return charges.CreditPurchaseCharge{}, fmt.Errorf("invalid credit purchase settlement type: %s", charge.Intent.Settlement.Type())
	}
}

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
