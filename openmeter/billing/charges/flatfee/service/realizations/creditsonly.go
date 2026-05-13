package realizations

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type AllocateCreditsOnlyInput struct {
	Charge             flatfee.Charge
	Amount             alpacadecimal.Decimal
	CurrencyCalculator currencyx.Calculator
}

func (i AllocateCreditsOnlyInput) Validate() error {
	if err := i.Charge.Validate(); err != nil {
		return fmt.Errorf("charge: %w", err)
	}

	if i.Amount.IsNegative() {
		return fmt.Errorf("amount cannot be negative")
	}

	if err := i.CurrencyCalculator.Validate(); err != nil {
		return fmt.Errorf("currency calculator: %w", err)
	}

	return nil
}

type AllocateCreditsOnlyResult struct {
	Allocated    alpacadecimal.Decimal
	Realizations creditrealization.Realizations
}

func (s *Service) AllocateCreditsOnly(ctx context.Context, in AllocateCreditsOnlyInput) (AllocateCreditsOnlyResult, error) {
	in.Amount = in.CurrencyCalculator.RoundToPrecision(in.Amount)

	if err := in.Validate(); err != nil {
		return AllocateCreditsOnlyResult{}, err
	}

	if in.Amount.IsZero() {
		return AllocateCreditsOnlyResult{}, nil
	}

	input := flatfee.OnCreditsOnlyUsageAccruedInput{
		Charge:           in.Charge,
		AmountToAllocate: in.Amount,
	}
	if err := input.Validate(); err != nil {
		return AllocateCreditsOnlyResult{}, fmt.Errorf("validate input: %w", err)
	}

	creditAllocations, err := s.handler.OnCreditsOnlyUsageAccrued(ctx, input)
	if err != nil {
		return AllocateCreditsOnlyResult{}, fmt.Errorf("on credits only usage accrued: %w", err)
	}

	allocated := in.CurrencyCalculator.RoundToPrecision(creditAllocations.Sum())
	if !allocated.Equal(in.Amount) {
		return AllocateCreditsOnlyResult{}, models.NewGenericValidationError(
			fmt.Errorf("credit allocations do not match total [charge_id=%s, total=%s, allocations_sum=%s]",
				in.Charge.ID, in.Amount.String(), allocated.String()),
		)
	}

	result := AllocateCreditsOnlyResult{
		Allocated: allocated,
	}

	if len(creditAllocations) > 0 {
		if in.Charge.Realizations.CurrentRun == nil {
			return AllocateCreditsOnlyResult{}, fmt.Errorf("current run is required")
		}

		realizations, err := s.createCreditAllocations(ctx, in.Charge, in.Charge.Realizations.CurrentRun.ID, creditAllocations.AsCreateInputs())
		if err != nil {
			return AllocateCreditsOnlyResult{}, fmt.Errorf("create credit allocations: %w", err)
		}

		if _, err := s.adapter.UpdateRealizationRun(ctx, flatfee.UpdateRealizationRunInput{
			ID: in.Charge.Realizations.CurrentRun.ID,
			Totals: mo.Some(totals.Totals{
				Amount:       allocated,
				CreditsTotal: allocated,
				Total:        alpacadecimal.Zero,
			}),
		}); err != nil {
			return AllocateCreditsOnlyResult{}, fmt.Errorf("update credit-only run totals: %w", err)
		}

		result.Realizations = realizations
	}

	return result, nil
}
