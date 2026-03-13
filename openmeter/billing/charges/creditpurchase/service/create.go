package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *service) Create(ctx context.Context, input creditpurchase.CreateInput) (creditpurchase.Charge, error) {
	if err := input.Validate(); err != nil {
		return creditpurchase.Charge{}, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (creditpurchase.Charge, error) {
		// Let's create the credit purchase charge
		charge, err := s.adapter.CreateCharge(ctx, creditpurchase.CreateChargeInput(input))
		if err != nil {
			return creditpurchase.Charge{}, err
		}

		// Let's active the state machine for the credit purchase charge
		switch charge.Intent.Settlement.Type() {
		case creditpurchase.SettlementTypePromotional:
			return s.onPromotionalCreditPurchase(ctx, charge)
		case creditpurchase.SettlementTypeInvoice:
			return creditpurchase.Charge{}, fmt.Errorf("invoice credit purchase settlements are not supported: %w", meta.ErrUnsupported)
		case creditpurchase.SettlementTypeExternal:
			return s.onExternalCreditPurchase(ctx, charge)
		default:
			return creditpurchase.Charge{}, fmt.Errorf("invalid credit purchase settlement type: %s", charge.Intent.Settlement.Type())
		}
	})
}
