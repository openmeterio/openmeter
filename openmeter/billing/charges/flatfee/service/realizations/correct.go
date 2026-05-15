package realizations

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// ReconcileCreditRealizationsInput describes the desired credit allocation
// total for a run whose billable amount changed while the invoice line stayed
// mutable.
type ReconcileCreditRealizationsInput struct {
	Charge             flatfee.Charge
	Run                flatfee.RealizationRun
	AllocateAt         time.Time
	TargetAmount       alpacadecimal.Decimal
	CurrencyCalculator currencyx.Calculator
}

func (i ReconcileCreditRealizationsInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if err := i.Run.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("run: %w", err))
	}

	if i.AllocateAt.IsZero() {
		errs = append(errs, errors.New("allocate at is required"))
	}

	if i.TargetAmount.IsNegative() {
		errs = append(errs, errors.New("target amount must be zero or positive"))
	}

	if err := i.CurrencyCalculator.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency calculator: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type ReconcileCreditRealizationsResult struct {
	Delta        alpacadecimal.Decimal
	Realizations creditrealization.Realizations
}

// ReconcileCredits adjusts a run's credit realizations to match TargetAmount.
//
// A positive delta allocates additional credits through the flat-fee handler.
// A negative delta creates credit corrections against the existing allocation
// lineage. A zero delta deliberately does nothing, because the current run is
// already backed by the right amount of credit realization rows.
func (s *Service) ReconcileCredits(ctx context.Context, in ReconcileCreditRealizationsInput) (ReconcileCreditRealizationsResult, error) {
	in.TargetAmount = in.CurrencyCalculator.RoundToPrecision(in.TargetAmount)

	if err := in.Validate(); err != nil {
		return ReconcileCreditRealizationsResult{}, err
	}

	currentAmount := in.CurrencyCalculator.RoundToPrecision(in.Run.CreditRealizations.Sum())
	delta := in.CurrencyCalculator.RoundToPrecision(in.TargetAmount.Sub(currentAmount))

	result := ReconcileCreditRealizationsResult{
		Delta: delta,
	}

	switch {
	case delta.IsPositive():
		// The mutable standard line grew, so the run needs extra credit
		// allocations for the new amount.
		handlerInput := flatfee.OnAllocateCreditsInput{
			Charge:                 in.Charge,
			ServicePeriod:          in.Run.ServicePeriod,
			PreTaxAmountToAllocate: delta,
		}
		if err := handlerInput.Validate(); err != nil {
			return ReconcileCreditRealizationsResult{}, fmt.Errorf("validating allocate credits input: %w", err)
		}

		creditAllocations, err := s.handler.OnAllocateCredits(ctx, handlerInput)
		if err != nil {
			return ReconcileCreditRealizationsResult{}, fmt.Errorf("allocate credits for flat fee: %w", err)
		}

		creditAllocationsWithLineID := creditrealization.CreateAllocationInputs(lo.Map(creditAllocations, func(allocation creditrealization.CreateAllocationInput, _ int) creditrealization.CreateAllocationInput {
			allocation.LineID = in.Run.LineID
			return allocation
		}))

		if len(creditAllocationsWithLineID) > 0 {
			realizations, err := s.createCreditAllocations(ctx, in.Charge, in.Run.ID, creditAllocationsWithLineID.AsCreateInputs())
			if err != nil {
				return ReconcileCreditRealizationsResult{}, fmt.Errorf("create credit allocations: %w", err)
			}

			result.Realizations = realizations
		}
	case delta.IsNegative():
		// The mutable standard line shrank. Correct the existing credit
		// realization lineage instead of creating unrelated negative rows.
		realizationIDs := lo.Map(in.Run.CreditRealizations, func(realization creditrealization.Realization, _ int) string {
			return realization.ID
		})
		lineageSegmentsByRealization, err := s.lineage.LoadActiveSegmentsByRealizationID(ctx, in.Charge.Namespace, realizationIDs)
		if err != nil {
			return ReconcileCreditRealizationsResult{}, fmt.Errorf("load active lineage segments for run: %w", err)
		}

		corrections, err := in.Run.CreditRealizations.Correct(
			delta,
			in.CurrencyCalculator,
			func(req creditrealization.CorrectionRequest) (creditrealization.CreateCorrectionInputs, error) {
				return s.handler.OnCorrectCreditAllocations(ctx, flatfee.CorrectCreditAllocationsInput{
					Charge:                       in.Charge,
					AllocateAt:                   in.AllocateAt,
					Corrections:                  req,
					LineageSegmentsByRealization: lineageSegmentsByRealization,
				})
			},
		)
		if err != nil {
			return ReconcileCreditRealizationsResult{}, fmt.Errorf("correct credits for run %s: %w", in.Run.ID.ID, err)
		}

		if len(corrections) > 0 {
			realizations, err := s.createCreditAllocations(ctx, in.Charge, in.Run.ID, corrections)
			if err != nil {
				return ReconcileCreditRealizationsResult{}, fmt.Errorf("create credit corrections for run %s: %w", in.Run.ID.ID, err)
			}

			result.Realizations = realizations
		}
	case delta.IsZero():
	}

	return result, nil
}

type CorrectAllCreditRealizationsInput struct {
	Charge             flatfee.Charge
	Run                flatfee.RealizationRun
	AllocateAt         time.Time
	CurrencyCalculator currencyx.Calculator
}

func (i CorrectAllCreditRealizationsInput) Validate() error {
	if err := i.Charge.Validate(); err != nil {
		return fmt.Errorf("charge: %w", err)
	}

	if err := i.Run.Validate(); err != nil {
		return fmt.Errorf("run: %w", err)
	}

	if i.AllocateAt.IsZero() {
		return fmt.Errorf("allocate at is required")
	}

	if err := i.CurrencyCalculator.Validate(); err != nil {
		return fmt.Errorf("currency calculator: %w", err)
	}

	return nil
}

type CorrectAllCreditRealizationsResult struct {
	Realizations creditrealization.Realizations
}

func (s *Service) CorrectAllCredits(ctx context.Context, in CorrectAllCreditRealizationsInput) (CorrectAllCreditRealizationsResult, error) {
	if err := in.Validate(); err != nil {
		return CorrectAllCreditRealizationsResult{}, err
	}

	realizationIDs := lo.Map(in.Run.CreditRealizations, func(realization creditrealization.Realization, _ int) string {
		return realization.ID
	})
	lineageSegmentsByRealization, err := s.lineage.LoadActiveSegmentsByRealizationID(ctx, in.Charge.Namespace, realizationIDs)
	if err != nil {
		return CorrectAllCreditRealizationsResult{}, fmt.Errorf("load active lineage segments: %w", err)
	}

	corrections, err := in.Run.CreditRealizations.CorrectAll(in.CurrencyCalculator, func(req creditrealization.CorrectionRequest) (creditrealization.CreateCorrectionInputs, error) {
		return s.handler.OnCorrectCreditAllocations(ctx, flatfee.CorrectCreditAllocationsInput{
			Charge:                       in.Charge,
			AllocateAt:                   in.AllocateAt,
			Corrections:                  req,
			LineageSegmentsByRealization: lineageSegmentsByRealization,
		})
	})
	if err != nil {
		return CorrectAllCreditRealizationsResult{}, fmt.Errorf("correct credits: %w", err)
	}

	result := CorrectAllCreditRealizationsResult{}
	if len(corrections) > 0 {
		realizations, err := s.createCreditAllocations(ctx, in.Charge, in.Run.ID, corrections)
		if err != nil {
			return CorrectAllCreditRealizationsResult{}, fmt.Errorf("create credit corrections: %w", err)
		}

		result.Realizations = realizations
	}

	return result, nil
}
