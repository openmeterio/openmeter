package run

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditreconciliation"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedrating "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type ReconcileRatedRunInput struct {
	Charge             usagebased.Charge
	Run                usagebased.RealizationRun
	Rating             usagebasedrating.GetDetailedRatingForUsageResult
	CurrencyCalculator currencyx.Currency
}

func (i ReconcileRatedRunInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if err := i.Run.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("run: %w", err))
	}

	if i.Charge.State.CurrentRealizationRunID == nil {
		errs = append(errs, errors.New("charge has no current realization run"))
	} else if *i.Charge.State.CurrentRealizationRunID != i.Run.ID.ID {
		errs = append(errs, fmt.Errorf(
			"run is not the charge's current realization run [current_run_id=%s run_id=%s]",
			*i.Charge.State.CurrentRealizationRunID,
			i.Run.ID.ID,
		))
	}

	if err := i.Rating.Totals.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("rating totals: %w", err))
	}

	if !i.Rating.Totals.CreditsTotal.IsZero() {
		errs = append(errs, errors.New("rating totals must not include credits"))
	}

	if err := i.Rating.DetailedLines.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("rating detailed lines: %w", err))
	}

	if i.Rating.Quantity.IsNegative() {
		errs = append(errs, errors.New("rating quantity must be zero or positive"))
	}

	if i.CurrencyCalculator == nil {
		errs = append(errs, errors.New("currency calculator is required"))
	} else if err := i.CurrencyCalculator.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency calculator: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type ReconcileRatedRunResult struct {
	Charge usagebased.Charge
	Run    usagebased.RealizationRun
}

// ReconcileRatedRun makes a realization run match an authoritative rating
// snapshot. Rating owns charge-currency reconciliation; settlement-fiat
// overage credits are reconciled later during invoice finalization.
func (s *Service) ReconcileRatedRun(
	ctx context.Context,
	in ReconcileRatedRunInput,
) (ReconcileRatedRunResult, error) {
	if err := in.Validate(); err != nil {
		return ReconcileRatedRunResult{}, err
	}

	run := in.Run
	runTotals := in.Rating.Totals.RoundToPrecision(in.CurrencyCalculator)
	settlementMode := in.Charge.Intent.GetSettlementMode()
	isCreditsOnlySettlementMode := settlementMode == productcatalog.CreditOnlySettlementMode
	creditReconciliationHandlerInput := CreditReconciliationHandlerInput{
		Charge:     in.Charge,
		Run:        run,
		AllocateAt: run.ServicePeriodTo,
	}

	reconcileResult, err := creditreconciliation.Reconcile(ctx, creditreconciliation.ReconcileInput{
		TargetAmount:    runTotals.Total,
		ExactAllocation: isCreditsOnlySettlementMode,
		Handler:         s.NewChargeCurrencyCreditReconciliationHandler(creditReconciliationHandlerInput),
	})
	if err != nil {
		return ReconcileRatedRunResult{}, fmt.Errorf("reconcile charge currency credits: %w", err)
	}

	run.CreditsAllocated = append(run.CreditsAllocated, reconcileResult.Realizations...)
	allocated := in.CurrencyCalculator.RoundToPrecision(run.CreditsAllocated.Sum())
	if allocated.GreaterThan(runTotals.Total) {
		return ReconcileRatedRunResult{}, fmt.Errorf(
			"credit allocations exceed rated total [charge_id=%s total=%s allocated=%s]",
			in.Charge.ID,
			runTotals.Total,
			allocated,
		)
	}

	runTotals.CreditsTotal = allocated
	runTotals.Total = in.CurrencyCalculator.RoundToPrecision(runTotals.Total.Sub(allocated))

	noFiatTransactionRequired := isCreditsOnlySettlementMode || runTotals.Total.IsZero()
	if settlementMode == productcatalog.CreditThenInvoiceSettlementMode && in.Charge.Intent.GetCurrency().IsCustom() {
		fiatOverage, err := in.Charge.ConvertCustomCurrencyOverageToFiat(runTotals)
		if err != nil {
			return ReconcileRatedRunResult{}, fmt.Errorf("convert custom currency overage to fiat: %w", err)
		}

		// Rating can decide that no invoice line is required for a zero gross
		// overage. For a positive overage the final value is set only after
		// settlement-fiat credit reconciliation during invoice finalization.
		noFiatTransactionRequired = fiatOverage.Amount.IsZero()
	}

	run.Totals = runTotals
	detailedLines := in.Rating.DetailedLines
	detailedLinesIncludeCreditAllocations := settlementMode == productcatalog.CreditThenInvoiceSettlementMode
	if detailedLinesIncludeCreditAllocations {
		creditsApplied, err := run.CreditsAllocated.AsCreditsApplied()
		if err != nil {
			return ReconcileRatedRunResult{}, fmt.Errorf("map credit realizations to detailed line credits: %w", err)
		}

		detailedLines, err = detailedLines.WithCreditsApplied(creditsApplied, in.CurrencyCalculator)
		if err != nil {
			return ReconcileRatedRunResult{}, fmt.Errorf("apply credit allocations to run detailed lines: %w", err)
		}
	}

	if err := s.adapter.UpsertRunDetailedLines(ctx, usagebased.UpsertRunDetailedLinesInput{
		ChargeID:                              in.Charge.GetChargeID(),
		RunID:                                 run.ID,
		DetailedLines:                         detailedLines,
		DetailedLinesIncludeCreditAllocations: detailedLinesIncludeCreditAllocations,
	}); err != nil {
		return ReconcileRatedRunResult{}, fmt.Errorf("upsert run detailed lines: %w", err)
	}
	run.DetailedLines = mo.Some(detailedLines)
	run.DetailedLinesIncludeCreditAllocations = detailedLinesIncludeCreditAllocations

	runBase, err := s.adapter.UpdateRealizationRun(ctx, usagebased.UpdateRealizationRunInput{
		ID:                        run.ID,
		StoredAtLT:                mo.Some(run.StoredAtLT),
		MeteredQuantity:           mo.Some(in.Rating.Quantity),
		Totals:                    mo.Some(runTotals),
		NoFiatTransactionRequired: mo.Some(noFiatTransactionRequired),
	})
	if err != nil {
		return ReconcileRatedRunResult{}, fmt.Errorf("update realization run: %w", err)
	}

	run.RealizationRunBase = runBase
	charge := in.Charge
	if err := charge.Realizations.SetRealizationRun(run); err != nil {
		return ReconcileRatedRunResult{}, fmt.Errorf("update realization run in charge: %w", err)
	}

	return ReconcileRatedRunResult{
		Charge: charge,
		Run:    run,
	}, nil
}
