package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

var _ charges.CreditPurchaseService = (*service)(nil)

func (s *service) UpdateExternalCreditPurchasePaymentState(ctx context.Context, input charges.UpdateExternalCreditPurchasePaymentStateInput) (charges.CreditPurchaseCharge, error) {
	if err := input.Validate(); err != nil {
		return charges.CreditPurchaseCharge{}, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (charges.CreditPurchaseCharge, error) {
		charge, err := s.adapter.GetChargeByID(ctx, charges.GetChargeByIDInput{
			ChargeID: input.ChargeID,
			Expands: charges.Expands{
				charges.ExpandRealizations,
			},
		})
		if err != nil {
			return charges.CreditPurchaseCharge{}, err
		}

		creditPurchaseCharge, err := charge.AsCreditPurchaseCharge()
		if err != nil {
			return charges.CreditPurchaseCharge{}, err
		}

		switch input.TargetPaymentState {
		case charges.PaymentSettlementStatusAuthorized:
			return s.creditPurchaseOrchestrator.HandleExternalCreditPurchasePaymentAuthorized(ctx, creditPurchaseCharge)
		case charges.PaymentSettlementStatusSettled:
			return s.creditPurchaseOrchestrator.HandleExternalCreditPurchasePaymentSettled(ctx, creditPurchaseCharge)
		default:
			return charges.CreditPurchaseCharge{}, fmt.Errorf("invalid target payment state: %s", input.TargetPaymentState)
		}
	})
}
