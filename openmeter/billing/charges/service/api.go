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
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/models"
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

	if currency.IsCustom() {
		return charges.Charge{}, models.NewGenericValidationError(fmt.Errorf("currency: %w", meta.ErrCustomCurrencyNotSupported))
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
			FeatureID:           input.FlatFee.FeatureID,
			SettlementMode:      input.FlatFee.SettlementMode,
		})
	case input.UsageBased != nil:
		chargeIntent = charges.NewChargeIntent(usagebased.Intent{
			Intent:              intent,
			IntentMutableFields: input.UsageBased.IntentMutableFields,
			FeatureID:           input.UsageBased.FeatureID,
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

func (s *service) DeleteCustomerCharge(ctx context.Context, input charges.DeleteCustomerChargeInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	if err := s.validateNamespaceLockdown(input.Namespace); err != nil {
		return err
	}

	policy, err := resolveDeletePolicy(input.PaymentAdjustment)
	if err != nil {
		return err
	}

	patch, err := meta.NewPatchDelete(meta.NewPatchDeleteInput{
		ChangeSource: billing.ChangeSourceAPIRequest,
		Policy:       policy,
	})
	if err != nil {
		return fmt.Errorf("creating charge delete patch: %w", err)
	}

	return s.ApplyPatches(ctx, charges.ApplyPatchesInput{
		CustomerID: customer.CustomerID{
			Namespace: input.Namespace,
			ID:        input.CustomerID,
		},
		PatchesByChargeID: map[string]charges.Patch{
			input.ChargeID: patch,
		},
	})
}

func resolveDeletePolicy(adjustment charges.PaymentAdjustment) (meta.PatchDeletePolicy, error) {
	switch adjustment {
	case charges.PaymentAdjustmentNone:
		return meta.PatchDeletePolicy{
			CreditRefundPolicy:  meta.CreditRefundPolicyIgnore,
			InvoiceRefundPolicy: meta.InvoiceRefundPolicyIgnore,
		}, nil
	default:
		return meta.PatchDeletePolicy{}, fmt.Errorf("unsupported payment adjustment: %s", adjustment)
	}
}
