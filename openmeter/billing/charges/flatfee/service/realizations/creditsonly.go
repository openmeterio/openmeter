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
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

type AllocateCreditsOnlyInput struct {
	Charge             flatfee.Charge
	Totals             totals.Totals
	CurrencyCalculator currencyx.Currency
}

func (i AllocateCreditsOnlyInput) Validate() error {
	if err := i.Charge.Validate(); err != nil {
		return fmt.Errorf("charge: %w", err)
	}

	if err := i.Totals.Validate(); err != nil {
		return fmt.Errorf("totals: %w", err)
	}

	if i.CurrencyCalculator == nil {
		return fmt.Errorf("currency calculator is required")
	}

	if err := i.CurrencyCalculator.Validate(); err != nil {
		return fmt.Errorf("currency calculator: %w", err)
	}

	return nil
}

type AllocateCreditsOnlyResult struct {
	Allocated          alpacadecimal.Decimal
	CreditRealizations creditrealization.Realizations
	RunBase            flatfee.RealizationRunBase
}

func (s *Service) AllocateCreditsOnly(ctx context.Context, in AllocateCreditsOnlyInput) (AllocateCreditsOnlyResult, error) {
	if err := in.Validate(); err != nil {
		return AllocateCreditsOnlyResult{}, err
	}

	amountToAllocate := in.CurrencyCalculator.RoundToPrecision(in.Totals.Total)

	servicePeriod := in.Charge.Intent.GetEffectiveServicePeriod()
	var creditAllocations creditrealization.CreateAllocationInputs
	if !amountToAllocate.IsZero() {
		input := flatfee.OnAllocateCreditsInput{
			Charge:                 in.Charge,
			ServicePeriod:          servicePeriod,
			BookedAt:               flatfee.UsageBookedAt(in.Charge.Intent.GetEffectivePaymentTerm(), servicePeriod),
			PreTaxAmountToAllocate: amountToAllocate,
		}
		if err := input.Validate(); err != nil {
			return AllocateCreditsOnlyResult{}, fmt.Errorf("validate input: %w", err)
		}

		var err error
		creditAllocations, err = s.handler.OnAllocateCredits(ctx, input)
		if err != nil {
			return AllocateCreditsOnlyResult{}, fmt.Errorf("allocate credits: %w", err)
		}
	}

	allocated := in.CurrencyCalculator.RoundToPrecision(creditAllocations.Sum())
	if !allocated.Equal(amountToAllocate) {
		return AllocateCreditsOnlyResult{}, models.NewGenericValidationError(
			fmt.Errorf("credit allocations do not match total [charge_id=%s, total=%s, allocations_sum=%s]",
				in.Charge.ID, amountToAllocate.String(), allocated.String()),
		)
	}

	if in.Charge.Realizations.CurrentRun == nil {
		return AllocateCreditsOnlyResult{}, fmt.Errorf("current run is required")
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (AllocateCreditsOnlyResult, error) {
		var realizations creditrealization.Realizations
		if len(creditAllocations) > 0 {
			var err error
			realizations, err = s.createCreditAllocations(ctx, in.Charge, in.Charge.Realizations.CurrentRun.ID, creditAllocations.AsCreateInputs())
			if err != nil {
				return AllocateCreditsOnlyResult{}, fmt.Errorf("create credit allocations: %w", err)
			}
		}

		runTotals := in.Totals
		runTotals.CreditsTotal = in.CurrencyCalculator.RoundToPrecision(runTotals.CreditsTotal.Add(allocated))
		runTotals.Total = in.CurrencyCalculator.RoundToPrecision(runTotals.Total.Sub(allocated))

		runBase, err := s.adapter.UpdateRealizationRun(ctx, flatfee.UpdateRealizationRunInput{
			ID:                        in.Charge.Realizations.CurrentRun.ID,
			Totals:                    mo.Some(runTotals),
			NoFiatTransactionRequired: mo.Some(true),
		})
		if err != nil {
			return AllocateCreditsOnlyResult{}, fmt.Errorf("update credit-only run totals: %w", err)
		}

		return AllocateCreditsOnlyResult{
			Allocated:          allocated,
			CreditRealizations: realizations,
			RunBase:            runBase,
		}, nil
	})
}
