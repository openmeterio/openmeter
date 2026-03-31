package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
)

type createRealizationRunInput struct {
	Run               usagebased.CreateRealizationRunInput
	CreditAllocations creditrealization.CreateAllocationInputs
}

func (i createRealizationRunInput) Validate() error {
	if err := i.Run.Validate(); err != nil {
		return fmt.Errorf("run: %w", err)
	}

	if err := i.CreditAllocations.Validate(); err != nil {
		return fmt.Errorf("credit allocations: %w", err)
	}

	return nil
}

func (s *service) createNewRealizationRun(ctx context.Context, charge usagebased.Charge, in createRealizationRunInput) (usagebased.Charge, error) {
	if err := in.Validate(); err != nil {
		return usagebased.Charge{}, err
	}

	if charge.State.CurrentRealizationRunID != nil {
		return usagebased.Charge{}, fmt.Errorf("current realization run already exists [charge_id=%s]", charge.GetChargeID())
	}

	run, err := s.adapter.CreateRealizationRun(ctx, charge.GetChargeID(), in.Run)
	if err != nil {
		return usagebased.Charge{}, fmt.Errorf("create realization run: %w", err)
	}

	creditRealizationCreates, err := in.CreditAllocations.AsAdapterCreateInputs()
	if err != nil {
		return usagebased.Charge{}, fmt.Errorf("as adapter create inputs: %w", err)
	}

	creditRealizations, err := s.adapter.CreateRunCreditAllocations(ctx, run.ID, creditRealizationCreates)
	if err != nil {
		return usagebased.Charge{}, fmt.Errorf("create credit allocations: %w", err)
	}

	charge.Realizations = append(charge.Realizations, usagebased.RealizationRun{
		RealizationRunBase: run,
		CreditsAllocated:   creditRealizations,
	})

	charge.State.CurrentRealizationRunID = lo.ToPtr(run.ID.ID)

	updatedCharge, err := s.adapter.UpdateCharge(ctx, charge.ChargeBase)
	if err != nil {
		return usagebased.Charge{}, fmt.Errorf("update charge: %w", err)
	}

	charge.ChargeBase = updatedCharge

	return charge, nil
}
