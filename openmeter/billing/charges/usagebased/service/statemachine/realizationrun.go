package statemachine

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedrating "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/samber/lo"
)

type StartRealizationRunInput struct {
	Type usagebased.RealizationRunType
}

func (i StartRealizationRunInput) Validate() error {
	if err := i.Type.Validate(); err != nil {
		return fmt.Errorf("type: %w", err)
	}

	return nil
}

func (s *Base) StartRealizationRun(ctx context.Context, in StartRealizationRunInput) error {
	if err := in.Validate(); err != nil {
		return fmt.Errorf("validate input: %w", err)
	}

	storedAtOffset := meta.NormalizeTimestamp(clock.Now().Add(-usagebased.InternalCollectionPeriod))
	collectionEnd, err := s.GetCollectionPeriodEnd(ctx)
	if err != nil {
		return fmt.Errorf("get collection period end: %w", err)
	}

	ratingResult, err := s.RatingService.GetRatingForUsage(ctx, usagebasedrating.GetRatingForUsageInput{
		Charge:         s.Charge,
		Customer:       s.CustomerOverride,
		FeatureMeter:   s.FeatureMeter,
		StoredAtOffset: storedAtOffset,
	})
	if err != nil {
		return fmt.Errorf("get rating for usage: %w", err)
	}

	totals := ratingResult.Totals.RoundToPrecision(s.CurrencyCalculator)

	if totals.Total.IsNegative() {
		return usagebased.ErrChargeTotalIsNegative.
			WithAttrs(models.Attributes{
				"total":     totals.Total.String(),
				"charge_id": s.Charge.ID,
			})
	}

	updatedCharge, err := s.createNewRealizationRun(ctx, s.Charge, usagebased.CreateRealizationRunInput{
		FeatureID:     s.Charge.State.FeatureID,
		Type:          usagebased.RealizationRunTypeFinalRealization,
		AsOf:          storedAtOffset,
		CollectionEnd: collectionEnd,
		MeterValue:    ratingResult.Quantity,
		Totals:        totals,
	})
	if err != nil {
		return fmt.Errorf("create new realization run: %w", err)
	}

	s.Charge = updatedCharge

	currentRun, err := updatedCharge.GetCurrentRealizationRun()
	if err != nil {
		return err
	}

	var creditRealizations creditrealization.Realizations
	if !totals.Total.IsZero() {
		creditAllocations, err := s.allocateCredits(ctx,
			usagebased.CreditsOnlyUsageAccruedInput{
				Charge:           updatedCharge,
				Run:              currentRun,
				AllocateAt:       storedAtOffset,
				AmountToAllocate: totals.Total,
			},
		)
		if err != nil {
			return fmt.Errorf("allocate credits: %w", err)
		}

		if len(creditAllocations) > 0 {
			creditRealizations, err = s.createRunCreditRealizations(ctx, updatedCharge, currentRun.ID, creditAllocations)
			if err != nil {
				return fmt.Errorf("create credit allocations: %w", err)
			}
		}

		currentRun.CreditsAllocated = creditRealizations
	}

	// We have allocated the required amount from credits, so we need to update totals accordingly
	totals.CreditsTotal = totals.CreditsTotal.Add(totals.Total)
	totals.Total = alpacadecimal.Zero

	currentRunBase, err := s.Adapter.UpdateRealizationRun(ctx, usagebased.UpdateRealizationRunInput{
		ID:         currentRun.ID,
		AsOf:       storedAtOffset,
		MeterValue: ratingResult.Quantity,
		Totals:     totals,
	})
	if err != nil {
		return fmt.Errorf("update realization run: %w", err)
	}

	currentRun.RealizationRunBase = currentRunBase

	if err := s.Charge.Realizations.SetRealizationRun(currentRun); err != nil {
		return fmt.Errorf("update realization run: %w", err)
	}

	return nil
}

func (s *Base) FinalizeRealizationRun(ctx context.Context) error {
	if s.Charge.State.CurrentRealizationRunID == nil {
		return fmt.Errorf("no realization run in progress [charge_id=%s]", s.Charge.ID)
	}

	currentRun, err := s.Charge.Realizations.GetByID(*s.Charge.State.CurrentRealizationRunID)
	if err != nil {
		return fmt.Errorf("get current realization run: %w", err)
	}

	storedAtOffset := meta.NormalizeTimestamp(clock.Now().Add(-usagebased.InternalCollectionPeriod))

	ratingResult, err := s.RatingService.GetRatingForUsage(ctx, usagebasedrating.GetRatingForUsageInput{
		Charge:         s.Charge,
		Customer:       s.CustomerOverride,
		FeatureMeter:   s.FeatureMeter,
		StoredAtOffset: storedAtOffset,
	})
	if err != nil {
		return fmt.Errorf("get rating for usage: %w", err)
	}

	currentTotals := ratingResult.Totals.RoundToPrecision(s.CurrencyCalculator)
	currentTotals.CreditsTotal = currentTotals.CreditsTotal.Add(currentTotals.Total)
	currentTotals.Total = alpacadecimal.Zero

	additionalAmount := s.CurrencyCalculator.RoundToPrecision(currentTotals.CreditsTotal.Sub(currentRun.Totals.CreditsTotal))

	switch {
	case additionalAmount.IsPositive():
		creditAllocations, err := s.allocateCredits(ctx,
			usagebased.CreditsOnlyUsageAccruedInput{
				Charge:           s.Charge,
				Run:              currentRun,
				AllocateAt:       storedAtOffset,
				AmountToAllocate: additionalAmount,
			},
		)
		if err != nil {
			return fmt.Errorf("allocate cred its: %w", err)
		}

		if len(creditAllocations) > 0 {
			if _, err := s.createRunCreditRealizations(ctx, s.Charge, currentRun.ID, creditAllocations); err != nil {
				return fmt.Errorf("create credit allocations: %w", err)
			}
		}
	case additionalAmount.IsNegative():
		realizationIDs := lo.Map(currentRun.CreditsAllocated, func(realization creditrealization.Realization, _ int) string {
			return realization.ID
		})
		lineageSegmentsByRealization, err := s.Lineage.LoadActiveSegmentsByRealizationID(ctx, s.Charge.Namespace, realizationIDs)
		if err != nil {
			return fmt.Errorf("load active lineage segments for current run: %w", err)
		}

		corrections, err := currentRun.CreditsAllocated.Correct(
			additionalAmount,
			s.CurrencyCalculator,
			func(req creditrealization.CorrectionRequest) (creditrealization.CreateCorrectionInputs, error) {
				return s.Handler.OnCreditsOnlyUsageAccruedCorrection(ctx, usagebased.CreditsOnlyUsageAccruedCorrectionInput{
					Charge:                       s.Charge,
					Run:                          currentRun,
					AllocateAt:                   storedAtOffset,
					Corrections:                  req,
					LineageSegmentsByRealization: lineageSegmentsByRealization,
				})
			},
		)
		if err != nil {
			return fmt.Errorf("correct credits: %w", err)
		}

		if len(corrections) > 0 {
			if _, err := s.createRunCreditRealizations(ctx, s.Charge, currentRun.ID, corrections); err != nil {
				return fmt.Errorf("create credit corrections: %w", err)
			}
		}
	case additionalAmount.IsZero():
	}

	if _, err := s.Adapter.UpdateRealizationRun(ctx, usagebased.UpdateRealizationRunInput{
		ID:         currentRun.ID,
		AsOf:       storedAtOffset,
		MeterValue: ratingResult.Quantity,
		Totals:     currentTotals,
	}); err != nil {
		return fmt.Errorf("update realization run: %w", err)
	}

	s.Charge.State.CurrentRealizationRunID = nil
	if _, err := s.Adapter.UpdateCharge(ctx, s.Charge.ChargeBase); err != nil {
		return fmt.Errorf("update charge: %w", err)
	}

	if err := s.refetchCharge(ctx); err != nil {
		return fmt.Errorf("refetch charge: %w", err)
	}

	return nil
}

func (s *Base) createNewRealizationRun(ctx context.Context, charge usagebased.Charge, in usagebased.CreateRealizationRunInput) (usagebased.Charge, error) {
	if err := in.Validate(); err != nil {
		return usagebased.Charge{}, err
	}

	if charge.State.CurrentRealizationRunID != nil {
		return usagebased.Charge{}, fmt.Errorf("current realization run already exists [charge_id=%s]", charge.GetChargeID())
	}

	run, err := s.Adapter.CreateRealizationRun(ctx, charge.GetChargeID(), in)
	if err != nil {
		return usagebased.Charge{}, fmt.Errorf("create realization run: %w", err)
	}

	charge.Realizations = append(charge.Realizations, usagebased.RealizationRun{
		RealizationRunBase: run,
	})

	charge.State.CurrentRealizationRunID = lo.ToPtr(run.ID.ID)

	updatedCharge, err := s.Adapter.UpdateCharge(ctx, charge.ChargeBase)
	if err != nil {
		return usagebased.Charge{}, fmt.Errorf("update charge: %w", err)
	}

	charge.ChargeBase = updatedCharge

	return charge, nil
}
