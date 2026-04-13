package run

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *Service) createRunCreditRealizations(ctx context.Context, charge usagebased.Charge, runID usagebased.RealizationRunID, creditAllocations creditrealization.CreateInputs) (creditrealization.Realizations, error) {
	realizations, err := s.adapter.CreateRunCreditRealization(ctx, runID, creditAllocations)
	if err != nil {
		return nil, err
	}

	if err := s.lineage.CreateInitialLineages(ctx, lineage.CreateInitialLineagesInput{
		Namespace:    charge.Namespace,
		ChargeID:     charge.ID,
		CustomerID:   charge.Intent.CustomerID,
		Currency:     charge.Intent.Currency,
		Realizations: realizations,
	}); err != nil {
		return nil, fmt.Errorf("create initial credit realization lineages: %w", err)
	}

	if err := s.lineage.PersistCorrectionLineageSegments(ctx, lineage.PersistCorrectionLineageSegmentsInput{
		Namespace:    charge.Namespace,
		Realizations: realizations,
	}); err != nil {
		return nil, fmt.Errorf("persist correction lineage segments: %w", err)
	}

	return realizations, nil
}

type allocateCreditRealizationsInput struct {
	Charge             usagebased.Charge
	Run                usagebased.RealizationRun
	AllocateAt         time.Time
	AmountToAllocate   alpacadecimal.Decimal
	CurrencyCalculator currencyx.Calculator
	Exact              bool
}

func (i allocateCreditRealizationsInput) Validate() error {
	if err := i.Charge.Validate(); err != nil {
		return fmt.Errorf("charge: %w", err)
	}

	if err := i.Run.Validate(); err != nil {
		return fmt.Errorf("run: %w", err)
	}

	if i.AllocateAt.IsZero() {
		return fmt.Errorf("allocate at is required")
	}

	if i.AmountToAllocate.IsNegative() {
		return fmt.Errorf("amount to allocate must be zero or positive")
	}

	if err := i.CurrencyCalculator.Validate(); err != nil {
		return fmt.Errorf("currency calculator: %w", err)
	}

	return nil
}

type allocateCreditRealizationsResult struct {
	Allocated    alpacadecimal.Decimal
	Realizations creditrealization.Realizations
}

func (s *Service) allocate(ctx context.Context, in allocateCreditRealizationsInput) (allocateCreditRealizationsResult, error) {
	in.AmountToAllocate = in.CurrencyCalculator.RoundToPrecision(in.AmountToAllocate)

	if err := in.Validate(); err != nil {
		return allocateCreditRealizationsResult{}, err
	}

	if in.AmountToAllocate.IsZero() {
		return allocateCreditRealizationsResult{}, nil
	}

	creditAllocations, err := s.handler.OnCreditsOnlyUsageAccrued(ctx, usagebased.CreditsOnlyUsageAccruedInput{
		Charge:           in.Charge,
		Run:              in.Run,
		AllocateAt:       in.AllocateAt,
		AmountToAllocate: in.AmountToAllocate,
	})
	if err != nil {
		return allocateCreditRealizationsResult{}, fmt.Errorf("on credits only usage accrued: %w", err)
	}

	allocated := in.CurrencyCalculator.RoundToPrecision(creditAllocations.Sum())
	if allocated.Sub(in.AmountToAllocate).IsPositive() {
		return allocateCreditRealizationsResult{}, usagebased.ErrCreditAllocationsDoNotMatchTotal.
			WithAttrs(models.Attributes{
				"total":     in.AmountToAllocate.String(),
				"charge_id": in.Charge.ID,
			})
	}

	if in.Exact && !allocated.Equal(in.AmountToAllocate) {
		return allocateCreditRealizationsResult{}, usagebased.ErrCreditAllocationsDoNotMatchTotal.
			WithAttrs(models.Attributes{
				"total":     in.AmountToAllocate.String(),
				"charge_id": in.Charge.ID,
			})
	}

	result := allocateCreditRealizationsResult{
		Allocated: allocated,
	}

	if len(creditAllocations) > 0 {
		realizations, err := s.createRunCreditRealizations(ctx, in.Charge, in.Run.ID, creditAllocations.AsCreateInputs())
		if err != nil {
			return allocateCreditRealizationsResult{}, fmt.Errorf("create credit allocations: %w", err)
		}

		result.Realizations = realizations
	}

	return result, nil
}
