package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

var _ charges.CreditPurchaseFacadeService = (*service)(nil)

func (s *service) HandleCreditPurchaseExternalPaymentStateTransition(ctx context.Context, input charges.HandleCreditPurchaseExternalPaymentStateTransitionInput) (creditpurchase.Charge, error) {
	if err := input.Validate(); err != nil {
		return creditpurchase.Charge{}, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (creditpurchase.Charge, error) {
		charge, err := s.GetByID(ctx, charges.GetByIDInput{
			ChargeID: input.ChargeID,
			Expands: meta.Expands{
				meta.ExpandRealizations,
			},
		})
		if err != nil {
			return creditpurchase.Charge{}, err
		}

		creditPurchaseCharge, err := charge.AsCreditPurchaseCharge()
		if err != nil {
			return creditpurchase.Charge{}, err
		}

		switch input.TargetPaymentState {
		case payment.StatusAuthorized:
			return s.creditPurchaseService.HandleExternalPaymentAuthorized(ctx, creditPurchaseCharge)
		case payment.StatusSettled:
			return s.creditPurchaseService.HandleExternalPaymentSettled(ctx, creditPurchaseCharge)
		default:
			return creditpurchase.Charge{}, fmt.Errorf("invalid target payment state: %s", input.TargetPaymentState)
		}
	})
}
