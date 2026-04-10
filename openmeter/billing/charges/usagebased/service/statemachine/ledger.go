package statemachine

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/lineage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *Base) allocateCredits(ctx context.Context, in usagebased.CreditsOnlyUsageAccruedInput) (creditrealization.CreateInputs, error) {
	in.AmountToAllocate = s.CurrencyCalculator.RoundToPrecision(in.AmountToAllocate)

	if err := in.Validate(); err != nil {
		return nil, err
	}

	creditAllocations, err := s.Handler.OnCreditsOnlyUsageAccrued(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("on credits only usage accrued: %w", err)
	}

	if in.Charge.Intent.SettlementMode == productcatalog.CreditOnlySettlementMode {
		if !creditAllocations.Sum().Equal(in.AmountToAllocate) {
			return nil, usagebased.ErrCreditAllocationsDoNotMatchTotal.
				WithAttrs(models.Attributes{
					"total":     in.AmountToAllocate.String(),
					"charge_id": in.Charge.ID,
				})
		}
	}

	return creditAllocations.AsCreateInputs(), nil
}

func (s *Base) createRunCreditRealizations(ctx context.Context, charge usagebased.Charge, runID usagebased.RealizationRunID, creditAllocations creditrealization.CreateInputs) (creditrealization.Realizations, error) {
	realizations, err := s.Adapter.CreateRunCreditRealization(ctx, runID, creditAllocations)
	if err != nil {
		return nil, err
	}

	// TODO: This is called for corrections too, we might be creating duplicates here.
	if err := s.Lineage.CreateInitialLineages(ctx, lineage.CreateInitialLineagesInput{
		Namespace:    charge.Namespace,
		ChargeID:     charge.ID,
		CustomerID:   charge.Intent.CustomerID,
		Currency:     charge.Intent.Currency,
		Realizations: realizations,
	}); err != nil {
		return nil, fmt.Errorf("create initial credit realization lineages: %w", err)
	}

	if err := s.Lineage.PersistCorrectionLineageSegments(ctx, lineage.PersistCorrectionLineageSegmentsInput{
		Namespace:    charge.Namespace,
		Realizations: realizations,
	}); err != nil {
		return nil, fmt.Errorf("persist correction lineage segments: %w", err)
	}

	return realizations, nil
}
