package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/currencies"
)

func (s *service) CreateCustomerCharge(ctx context.Context, input charges.CreateCustomerChargeInput) (charges.Charge, error) {
	if err := input.Validate(); err != nil {
		return charges.Charge{}, err
	}

	currency, err := s.currencyResolver.ResolveCurrency(ctx, input.Namespace, currencies.CurrencyRef{
		Code: input.CurrencyCode,
	})
	if err != nil {
		return charges.Charge{}, fmt.Errorf("resolving currency: %w", err)
	}

	intent := meta.Intent{
		ManagedBy:         billing.ManuallyManagedLine,
		CustomerID:        input.CustomerID,
		Currency:          *currency,
		TaxConfig:         input.TaxConfig,
		UniqueReferenceID: input.UniqueReferenceID,
	}

	var chargeIntent charges.ChargeIntent
	switch {
	case input.FlatFee != nil:
		chargeIntent = charges.NewChargeIntent(flatfee.Intent{
			Intent:              intent,
			IntentMutableFields: input.FlatFee.IntentMutableFields,
			FeatureKey:          input.FlatFee.FeatureKey,
			SettlementMode:      input.FlatFee.SettlementMode,
		})
	case input.UsageBased != nil:
		chargeIntent = charges.NewChargeIntent(usagebased.Intent{
			Intent:              intent,
			IntentMutableFields: input.UsageBased.IntentMutableFields,
			FeatureKey:          input.UsageBased.FeatureKey,
			SettlementMode:      input.UsageBased.SettlementMode,
		})
	}

	created, err := s.Create(ctx, charges.CreateInput{
		Namespace: input.Namespace,
		Intents:   charges.ChargeIntents{chargeIntent},
	})
	if err != nil {
		return charges.Charge{}, err
	}

	if len(created) != 1 {
		return charges.Charge{}, fmt.Errorf("expected one created charge, got %d", len(created))
	}

	return created[0], nil
}
