package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/models"
)

type resolveInitialCostBasisInput struct {
	Currency   currencies.Currency
	CostBasis  *creditpurchase.CostBasis
	ResolvedAt time.Time
}

var _ models.Validator = (*resolveInitialCostBasisInput)(nil)

func (i resolveInitialCostBasisInput) Validate() error {
	var errs []error

	if err := i.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}
	if i.CostBasis != nil {
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
	if input.CostBasis == nil {
		return nil, nil
	}

	switch input.CostBasis.Type() {
	case creditpurchase.CostBasisTypeFiat:
		fiatCostBasis, err := input.CostBasis.AsFiat()
		if err != nil {
			return nil, fmt.Errorf("getting fiat cost basis: %w", err)
		}

		return lo.ToPtr(costbasis.State{
			CostBasis:  fiatCostBasis.Rate,
			ResolvedAt: input.ResolvedAt,
		}), nil
	case creditpurchase.CostBasisTypeCustomCurrency:
		customCostBasis, err := input.CostBasis.AsCustomCurrency()
		if err != nil {
			return nil, fmt.Errorf("getting custom-currency cost basis: %w", err)
		}

		if customCostBasis.Kind() == costbasis.ModeDynamic {
			return nil, errors.New("dynamic cost basis is not supported for credit purchases")
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
