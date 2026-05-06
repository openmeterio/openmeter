package run

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedrating "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type CreateRatedRunInput struct {
	Charge             usagebased.Charge
	CustomerOverride   billing.CustomerOverrideWithDetails
	FeatureMeter       feature.FeatureMeter
	Type               usagebased.RealizationRunType
	StoredAtLT         time.Time
	ServicePeriodTo    time.Time
	LineID             *string
	InvoiceID          *string
	CreditAllocation   CreditAllocationMode
	CurrencyCalculator currencyx.Calculator
}

func (i CreateRatedRunInput) Validate() error {
	if err := i.Charge.Validate(); err != nil {
		return fmt.Errorf("charge: %w", err)
	}

	if i.Charge.State.CurrentRealizationRunID != nil {
		return fmt.Errorf("charge: current realization run already exists [charge_id=%s]", i.Charge.GetChargeID())
	}

	if i.CustomerOverride.Customer == nil {
		return fmt.Errorf("expanded customer is required")
	}

	if i.FeatureMeter.Meter == nil {
		return fmt.Errorf("feature meter is required")
	}

	if err := i.Type.Validate(); err != nil {
		return fmt.Errorf("type: %w", err)
	}

	if i.StoredAtLT.IsZero() {
		return fmt.Errorf("stored at lt is required")
	}

	if i.ServicePeriodTo.IsZero() {
		return fmt.Errorf("service period to is required")
	}

	period := i.Charge.Intent.ServicePeriod
	if !i.ServicePeriodTo.After(period.From) {
		return fmt.Errorf("service period to must be after charge service period from")
	}

	if i.ServicePeriodTo.After(period.To) {
		return fmt.Errorf("service period to must not be after charge service period to")
	}

	if i.LineID != nil && *i.LineID == "" {
		return fmt.Errorf("line id if set, must be non-empty")
	}

	if i.InvoiceID != nil && *i.InvoiceID == "" {
		return fmt.Errorf("invoice id if set, must be non-empty")
	}

	if err := i.CreditAllocation.Validate(); err != nil {
		return fmt.Errorf("credit allocation: %w", err)
	}

	if err := i.CurrencyCalculator.Validate(); err != nil {
		return fmt.Errorf("currency calculator: %w", err)
	}

	return nil
}

type CreateRatedRunResult struct {
	Charge usagebased.Charge
	Run    usagebased.RealizationRun
	Rating usagebasedrating.GetDetailedRatingForUsageResult
}

func (s *Service) createNewRealizationRun(ctx context.Context, charge usagebased.Charge, in usagebased.CreateRealizationRunInput) (usagebased.Charge, error) {
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

func (s *Service) CreateRatedRun(ctx context.Context, in CreateRatedRunInput) (CreateRatedRunResult, error) {
	if err := in.Validate(); err != nil {
		return CreateRatedRunResult{}, err
	}

	ratingResult, err := s.rater.GetDetailedRatingForUsage(ctx, usagebasedrating.GetDetailedRatingForUsageInput{
		Charge:          in.Charge,
		StoredAtLT:      in.StoredAtLT,
		ServicePeriodTo: in.ServicePeriodTo,
		Customer:        in.CustomerOverride,
		FeatureMeter:    in.FeatureMeter,
	})
	if err != nil {
		return CreateRatedRunResult{}, fmt.Errorf("get detailed rating for usage: %w", err)
	}

	runTotals := ratingResult.Totals.RoundToPrecision(in.CurrencyCalculator)
	if runTotals.Total.IsNegative() {
		return CreateRatedRunResult{}, usagebased.ErrChargeTotalIsNegative.
			WithAttrs(models.Attributes{
				"total":     runTotals.Total.String(),
				"charge_id": in.Charge.ID,
			})
	}

	updatedCharge, err := s.createNewRealizationRun(ctx, in.Charge, usagebased.CreateRealizationRunInput{
		FeatureID:       in.Charge.State.FeatureID,
		Type:            in.Type,
		StoredAtLT:      in.StoredAtLT,
		ServicePeriodTo: in.ServicePeriodTo,
		LineID:          in.LineID,
		InvoiceID:       in.InvoiceID,
		MeteredQuantity: ratingResult.Quantity,
		Totals:          runTotals,
	})
	if err != nil {
		return CreateRatedRunResult{}, fmt.Errorf("create new realization run: %w", err)
	}

	currentRun, err := updatedCharge.GetCurrentRealizationRun()
	if err != nil {
		return CreateRatedRunResult{}, err
	}

	if err := s.adapter.UpsertRunDetailedLines(ctx, updatedCharge.GetChargeID(), currentRun.ID, ratingResult.DetailedLines); err != nil {
		return CreateRatedRunResult{}, fmt.Errorf("upsert run detailed lines: %w", err)
	}
	currentRun.DetailedLines = mo.Some(ratingResult.DetailedLines)

	if in.CreditAllocation != CreditAllocationNone {
		allocationResult, err := s.allocate(ctx, allocateCreditRealizationsInput{
			Charge:             updatedCharge,
			Run:                currentRun,
			AllocateAt:         in.StoredAtLT,
			AmountToAllocate:   runTotals.Total,
			CurrencyCalculator: in.CurrencyCalculator,
			Exact:              in.CreditAllocation == CreditAllocationExact,
		})
		if err != nil {
			return CreateRatedRunResult{}, fmt.Errorf("allocate credits: %w", err)
		}

		currentRun.CreditsAllocated = allocationResult.Realizations
		runTotals.CreditsTotal = runTotals.CreditsTotal.Add(allocationResult.Allocated)
		runTotals.Total = in.CurrencyCalculator.RoundToPrecision(runTotals.Total.Sub(allocationResult.Allocated))

		currentRunBase, err := s.adapter.UpdateRealizationRun(ctx, usagebased.UpdateRealizationRunInput{
			ID:     currentRun.ID,
			Totals: mo.Some(runTotals),
		})
		if err != nil {
			return CreateRatedRunResult{}, fmt.Errorf("update realization run: %w", err)
		}

		currentRun.RealizationRunBase = currentRunBase

		if err := updatedCharge.Realizations.SetRealizationRun(currentRun); err != nil {
			return CreateRatedRunResult{}, fmt.Errorf("update realization run: %w", err)
		}
	}

	if err := updatedCharge.Realizations.SetRealizationRun(currentRun); err != nil {
		return CreateRatedRunResult{}, fmt.Errorf("update realization run detailed lines: %w", err)
	}

	return CreateRatedRunResult{
		Charge: updatedCharge,
		Run:    currentRun,
		Rating: ratingResult,
	}, nil
}
