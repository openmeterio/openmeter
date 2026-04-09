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
		CustomerID:   charge.Intent.CustomerID,
		Currency:     charge.Intent.Currency,
		Realizations: realizations,
	}); err != nil {
		return nil, fmt.Errorf("create initial credit realization lineages: %w", err)
	}

	if err := s.lineage.WritebackCorrectionLineageSegments(ctx, lineage.WritebackCorrectionLineageSegmentsInput{
		Namespace:    charge.Namespace,
		Realizations: realizations,
	}); err != nil {
		return nil, fmt.Errorf("write back correction lineage segments: %w", err)
	}

	return realizations, nil
}
