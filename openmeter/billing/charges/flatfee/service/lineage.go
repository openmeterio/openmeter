package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
)

func (s *service) createCreditAllocations(ctx context.Context, charge flatfee.Charge, creditAllocations creditrealization.CreateInputs) (creditrealization.Realizations, error) {
	realizations, err := s.adapter.CreateCreditAllocations(ctx, charge.GetChargeID(), creditAllocations)
	if err != nil {
		return creditrealization.Realizations{}, err
	}

	if err := s.lineage.CreateInitialLineages(ctx, lineage.CreateInitialLineagesInput{
		Namespace:    charge.Namespace,
		CustomerID:   charge.Intent.CustomerID,
		Currency:     charge.Intent.Currency,
		Realizations: realizations,
	}); err != nil {
		return creditrealization.Realizations{}, fmt.Errorf("create initial credit realization lineages: %w", err)
	}

	if err := s.lineage.WritebackCorrectionLineageSegments(ctx, lineage.WritebackCorrectionLineageSegmentsInput{
		Namespace:    charge.Namespace,
		Realizations: realizations,
	}); err != nil {
		return creditrealization.Realizations{}, fmt.Errorf("write back correction lineage segments: %w", err)
	}

	return realizations, nil
}
