package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
)

func (s *service) createRunCreditRealizations(ctx context.Context, charge usagebased.Charge, runID usagebased.RealizationRunID, creditAllocations creditrealization.CreateInputs) (creditrealization.Realizations, error) {
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
