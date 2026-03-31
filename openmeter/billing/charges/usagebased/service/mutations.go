package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
)

func (s *service) createNewRealizationRun(ctx context.Context, charge usagebased.Charge, in usagebased.CreateRealizationRunInput) (usagebased.Charge, error) {
	if err := in.Validate(); err != nil {
		return usagebased.Charge{}, err
	}

	if charge.State.CurrentRealizationRunID != nil {
		return usagebased.Charge{}, fmt.Errorf("current realization run already exists [charge_id=%s]", charge.GetChargeID())
	}

	run, err := s.adapter.CreateRealizationRun(ctx, charge.GetChargeID(), in)
	if err != nil {
		return usagebased.Charge{}, fmt.Errorf("create realization run: %w", err)
	}

	charge.Realizations = append(charge.Realizations, usagebased.RealizationRun{
		RealizationRunBase: run,
	})

	charge.State.CurrentRealizationRunID = lo.ToPtr(run.ID.ID)

	updatedCharge, err := s.adapter.UpdateCharge(ctx, charge.ChargeBase)
	if err != nil {
		return usagebased.Charge{}, fmt.Errorf("update charge: %w", err)
	}

	charge.ChargeBase = updatedCharge

	return charge, nil
}
