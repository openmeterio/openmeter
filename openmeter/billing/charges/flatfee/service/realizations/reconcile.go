package realizations

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditreconciliation"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

type ReconcileRatedRunInput struct {
	Charge             flatfee.Charge
	Run                flatfee.RealizationRun
	Rating             RateResult
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

	if i.Charge.Realizations.CurrentRun == nil {
		errs = append(errs, errors.New("charge has no current realization run"))
	} else if i.Charge.Realizations.CurrentRun.ID != i.Run.ID {
		errs = append(errs, fmt.Errorf(
			"run is not the charge's current realization run [current_run_id=%s run_id=%s]",
			i.Charge.Realizations.CurrentRun.ID.ID,
			i.Run.ID.ID,
		))
	}

	if err := i.Rating.Intent.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("rating intent: %w", err))
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

	if i.CurrencyCalculator == nil {
		errs = append(errs, errors.New("currency calculator is required"))
	} else if err := i.CurrencyCalculator.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency calculator: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// ReconcileRatedRun makes a flat-fee realization run match an authoritative
// rating. Rating owns charge-currency reconciliation; settlement-fiat
// overage credits are allocated later during invoice finalization.
func (s *Service) ReconcileRatedRun(
	ctx context.Context,
	in ReconcileRatedRunInput,
) (flatfee.RealizationRun, error) {
	if err := in.Validate(); err != nil {
		return flatfee.RealizationRun{}, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (flatfee.RealizationRun, error) {
		run := in.Run
		run.ServicePeriod = in.Rating.Intent.ServicePeriod
		run.AmountAfterProration = in.Rating.Intent.AmountAfterProration
		runTotals := in.Rating.Totals.RoundToPrecision(in.CurrencyCalculator)
		settlementMode := in.Charge.Intent.GetSettlementMode()
		isCreditsOnlySettlementMode := settlementMode == productcatalog.CreditOnlySettlementMode
		creditReconciliationHandlerInput := CreditReconciliationHandlerInput{
			Charge: in.Charge,
			Run:    run,
			AllocateAt: flatfee.UsageBookedAt(
				in.Charge.Intent.GetEffectivePaymentTerm(),
				in.Rating.Intent.ServicePeriod,
			),
		}

		reconcileResult, err := creditreconciliation.Reconcile(ctx, creditreconciliation.ReconcileInput{
			TargetAmount:    runTotals.Total,
			ExactAllocation: isCreditsOnlySettlementMode,
			Handler:         s.NewChargeCurrencyCreditReconciliationHandler(creditReconciliationHandlerInput),
		})
		if err != nil {
			return flatfee.RealizationRun{}, fmt.Errorf("reconcile charge currency credits: %w", err)
		}

		run.CreditRealizations = append(run.CreditRealizations, reconcileResult.Realizations...)
		allocated := in.CurrencyCalculator.RoundToPrecision(run.CreditRealizations.Sum())
		if allocated.GreaterThan(runTotals.Total) {
			return flatfee.RealizationRun{}, fmt.Errorf(
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
				return flatfee.RealizationRun{}, fmt.Errorf("convert custom currency overage to fiat: %w", err)
			}

			// Rating can decide that no invoice line is required for a zero gross
			// overage. For a positive overage the final value is set only after
			// settlement-fiat credit allocation during invoice finalization.
			noFiatTransactionRequired = fiatOverage.Amount.IsZero()
		}

		detailedLines := in.Rating.DetailedLines
		if settlementMode == productcatalog.CreditThenInvoiceSettlementMode {
			creditsApplied, err := run.CreditRealizations.AsCreditsApplied()
			if err != nil {
				return flatfee.RealizationRun{}, fmt.Errorf("map credit realizations to detailed line credits: %w", err)
			}

			detailedLines, err = detailedLines.WithCreditsApplied(creditsApplied, in.CurrencyCalculator)
			if err != nil {
				return flatfee.RealizationRun{}, fmt.Errorf("apply credit allocations to run detailed lines: %w", err)
			}
		}

		if err := s.adapter.UpsertDetailedLines(ctx, run.ID, detailedLines); err != nil {
			return flatfee.RealizationRun{}, fmt.Errorf("persist detailed lines for run[%s]: %w", run.ID.ID, err)
		}

		runBase, err := s.adapter.UpdateRealizationRun(ctx, flatfee.UpdateRealizationRunInput{
			ID:                        run.ID,
			ServicePeriod:             mo.Some(in.Rating.Intent.ServicePeriod),
			AmountAfterProration:      mo.Some(in.Rating.Intent.AmountAfterProration),
			Totals:                    mo.Some(runTotals),
			NoFiatTransactionRequired: mo.Some(noFiatTransactionRequired),
		})
		if err != nil {
			return flatfee.RealizationRun{}, fmt.Errorf("update realization run[%s]: %w", run.ID.ID, err)
		}

		run.RealizationRunBase = runBase
		run.DetailedLines = mo.Some(detailedLines)

		return run, nil
	})
}
