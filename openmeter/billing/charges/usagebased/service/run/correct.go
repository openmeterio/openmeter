package run

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

type ReconcileCreditRealizationsInput struct {
	Charge             usagebased.Charge
	Run                usagebased.RealizationRun
	AllocateAt         time.Time
	TargetAmount       alpacadecimal.Decimal
	CurrencyCalculator currencyx.Calculator
	ExactAllocation    bool
}

func (i ReconcileCreditRealizationsInput) Validate() error {
	if err := i.Charge.Validate(); err != nil {
		return fmt.Errorf("charge: %w", err)
	}

	if err := i.Run.Validate(); err != nil {
		return fmt.Errorf("run: %w", err)
	}

	if i.AllocateAt.IsZero() {
		return fmt.Errorf("allocate at is required")
	}

	if i.TargetAmount.IsNegative() {
		return fmt.Errorf("target amount must be zero or positive")
	}

	if err := i.CurrencyCalculator.Validate(); err != nil {
		return fmt.Errorf("currency calculator: %w", err)
	}

	return nil
}

type ReconcileCreditRealizationsResult struct {
	Delta        alpacadecimal.Decimal
	Realizations creditrealization.Realizations
}

func (s *Service) ReconcileCredits(ctx context.Context, in ReconcileCreditRealizationsInput) (ReconcileCreditRealizationsResult, error) {
	in.TargetAmount = in.CurrencyCalculator.RoundToPrecision(in.TargetAmount)

	if err := in.Validate(); err != nil {
		return ReconcileCreditRealizationsResult{}, err
	}

	currentAmount := in.CurrencyCalculator.RoundToPrecision(in.Run.Totals.CreditsTotal)
	delta := in.CurrencyCalculator.RoundToPrecision(in.TargetAmount.Sub(currentAmount))

	result := ReconcileCreditRealizationsResult{
		Delta: delta,
	}

	switch {
	case delta.IsPositive():
		allocated, err := s.allocate(ctx, allocateCreditRealizationsInput{
			Charge:             in.Charge,
			Run:                in.Run,
			AllocateAt:         in.AllocateAt,
			AmountToAllocate:   delta,
			CurrencyCalculator: in.CurrencyCalculator,
			Exact:              in.ExactAllocation,
		})
		if err != nil {
			return ReconcileCreditRealizationsResult{}, err
		}

		result.Realizations = allocated.Realizations
	case delta.IsNegative():
		realizationIDs := lo.Map(in.Run.CreditsAllocated, func(realization creditrealization.Realization, _ int) string {
			return realization.ID
		})
		lineageSegmentsByRealization, err := s.lineage.LoadActiveSegmentsByRealizationID(ctx, in.Charge.Namespace, realizationIDs)
		if err != nil {
			return ReconcileCreditRealizationsResult{}, fmt.Errorf("load active lineage segments for run: %w", err)
		}

		corrections, err := in.Run.CreditsAllocated.Correct(
			delta,
			in.CurrencyCalculator,
			func(req creditrealization.CorrectionRequest) (creditrealization.CreateCorrectionInputs, error) {
				return s.handler.OnCreditsOnlyUsageAccruedCorrection(ctx, usagebased.CreditsOnlyUsageAccruedCorrectionInput{
					Charge:                       in.Charge,
					Run:                          in.Run,
					AllocateAt:                   in.AllocateAt,
					Corrections:                  req,
					LineageSegmentsByRealization: lineageSegmentsByRealization,
				})
			},
		)
		if err != nil {
			return ReconcileCreditRealizationsResult{}, fmt.Errorf("correct credits: %w", err)
		}

		if len(corrections) > 0 {
			realizations, err := s.createRunCreditRealizations(ctx, in.Charge, in.Run.ID, corrections)
			if err != nil {
				return ReconcileCreditRealizationsResult{}, fmt.Errorf("create credit corrections: %w", err)
			}

			result.Realizations = realizations
		}
	case delta.IsZero():
	}

	return result, nil
}

type CorrectAllCreditRealizationsInput struct {
	Charge             usagebased.Charge
	Run                usagebased.RealizationRun
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

	realizationIDs := lo.Map(in.Run.CreditsAllocated, func(realization creditrealization.Realization, _ int) string {
		return realization.ID
	})
	lineageSegmentsByRealization, err := s.lineage.LoadActiveSegmentsByRealizationID(ctx, in.Charge.Namespace, realizationIDs)
	if err != nil {
		return CorrectAllCreditRealizationsResult{}, fmt.Errorf("load active lineage segments for run: %w", err)
	}

	corrections, err := in.Run.CreditsAllocated.CorrectAll(in.CurrencyCalculator, func(req creditrealization.CorrectionRequest) (creditrealization.CreateCorrectionInputs, error) {
		return s.handler.OnCreditsOnlyUsageAccruedCorrection(ctx, usagebased.CreditsOnlyUsageAccruedCorrectionInput{
			Charge:                       in.Charge,
			Run:                          in.Run,
			AllocateAt:                   in.AllocateAt,
			Corrections:                  req,
			LineageSegmentsByRealization: lineageSegmentsByRealization,
		})
	})
	if err != nil {
		return CorrectAllCreditRealizationsResult{}, fmt.Errorf("correct credits for run %s: %w", in.Run.ID.ID, err)
	}

	result := CorrectAllCreditRealizationsResult{}
	if len(corrections) > 0 {
		realizations, err := s.createRunCreditRealizations(ctx, in.Charge, in.Run.ID, corrections)
		if err != nil {
			return CorrectAllCreditRealizationsResult{}, fmt.Errorf("create credit corrections for run %s: %w", in.Run.ID.ID, err)
		}

		result.Realizations = realizations
	}

	return result, nil
}
