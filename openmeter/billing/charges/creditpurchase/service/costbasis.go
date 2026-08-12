package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/models"
)

type resolveInitialCostBasisInput struct {
	Currency   currencies.Currency
	CostBasis  creditpurchase.CostBasis
	ResolvedAt time.Time
}

var _ models.Validator = (*resolveInitialCostBasisInput)(nil)

func (i resolveInitialCostBasisInput) Validate() error {
	var errs []error

	if err := i.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}
	if !i.CostBasis.IsEmpty() {
		if err := i.CostBasis.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("cost basis: %w", err))
		}
	}
	if i.ResolvedAt.IsZero() {
		errs = append(errs, errors.New("resolved at is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (s *service) resolveInitialCostBasis(ctx context.Context, input resolveInitialCostBasisInput) (*costbasis.State, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	if input.CostBasis.IsEmpty() {
		return nil, nil
	}

	switch input.CostBasis.Type() {
	case creditpurchase.CostBasisTypeFiat:
		// Fiat costbasis is resolved on read from the intent, it doesn't have a credit_purchase_cost_basis row.
		return nil, nil
	case creditpurchase.CostBasisTypeCustomCurrency:
		customCostBasis, err := input.CostBasis.AsCustomCurrency()
		if err != nil {
			return nil, fmt.Errorf("getting custom-currency cost basis: %w", err)
		}

		resolvedCostBasis, err := s.costbasisResolver.ResolveInitialState(ctx, costbasis.ResolveInitialStateInput{
			CurrencyID: input.Currency.NamespacedID,
			Intent:     customCostBasis,
			ResolvedAt: input.ResolvedAt,
		})
		if err != nil {
			return nil, fmt.Errorf("resolving cost basis: %w", err)
		}

		return resolvedCostBasis, nil
	default:
		return nil, fmt.Errorf("unsupported credit purchase cost basis type: %s", input.CostBasis.Type())
	}
}

// ResolveDynamicCostBasis pins the cost basis effective at the purchase's
// service-period start before the first monetary effect.
func (s *stateMachine) ResolveDynamicCostBasis(ctx context.Context) error {
	if s.Charge.Intent.CostBasis.Type() != creditpurchase.CostBasisTypeCustomCurrency {
		return nil
	}

	intent, err := s.Charge.Intent.CostBasis.AsCustomCurrency()
	if err != nil {
		return fmt.Errorf("getting custom-currency cost basis: %w", err)
	}

	if intent.Kind() != costbasis.ModeDynamic || s.Charge.State.ResolvedCostBasis != nil {
		return nil
	}

	if s.Charge.State.ChargeCostBasisID == nil {
		return models.NewGenericPreConditionFailedError(
			fmt.Errorf("charge cost basis reference is missing for credit purchase %s", s.Charge.ID),
		)
	}

	resolvedState, err := s.CostBasisResolver.ResolveDynamicState(ctx, costbasis.ResolveDynamicStateInput{
		CurrencyID:        s.Charge.Intent.Currency.NamespacedID,
		Intent:            intent,
		ServicePeriodFrom: s.Charge.Intent.ServicePeriod.From,
	})
	if err != nil {
		return fmt.Errorf("resolving dynamic cost basis for credit purchase %s: %w", s.Charge.ID, err)
	}

	persisted, err := s.Adapter.SetResolvedCostBasis(ctx, creditpurchase.SetResolvedCostBasisInput{
		ChargeID:          s.Charge.GetChargeID(),
		ChargeCostBasisID: *s.Charge.State.ChargeCostBasisID,
		State:             resolvedState,
	})
	if err != nil {
		return fmt.Errorf("persisting dynamic cost basis for credit purchase %s: %w", s.Charge.ID, err)
	}

	if persisted.State == nil {
		return fmt.Errorf("persisted dynamic cost basis is unresolved for credit purchase %s", s.Charge.ID)
	}

	s.Charge.State.ResolvedCostBasis = persisted.State

	return nil
}
