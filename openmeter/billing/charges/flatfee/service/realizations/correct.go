package realizations

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

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
		return s.handler.OnCreditsOnlyUsageAccruedCorrection(ctx, flatfee.CreditsOnlyUsageAccruedCorrectionInput{
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
