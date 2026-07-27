package realizations

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

type StartCreditThenInvoiceRunInput struct {
	Charge    flatfee.Charge
	LineID    string
	InvoiceID string
}

func (i StartCreditThenInvoiceRunInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if i.LineID == "" {
		errs = append(errs, errors.New("line ID is required"))
	}

	if i.InvoiceID == "" {
		errs = append(errs, errors.New("invoice ID is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type StartCreditThenInvoiceRunResult struct {
	Run flatfee.RealizationRun
}

func (s *Service) StartCreditThenInvoiceRun(ctx context.Context, in StartCreditThenInvoiceRunInput) (StartCreditThenInvoiceRunResult, error) {
	if err := in.Validate(); err != nil {
		return StartCreditThenInvoiceRunResult{}, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (StartCreditThenInvoiceRunResult, error) {
		currency := in.Charge.Intent.GetCurrency()

		rateableIntent, err := in.Charge.GetRateableIntent()
		if err != nil {
			return StartCreditThenInvoiceRunResult{}, fmt.Errorf("getting rateable intent: %w", err)
		}

		ratingResult, err := s.Rate(rateableIntent)
		if err != nil {
			return StartCreditThenInvoiceRunResult{}, fmt.Errorf("rating flat fee: %w", err)
		}

		runBase, err := s.adapter.CreateCurrentRun(ctx, flatfee.CreateCurrentRunInput{
			Charge:                    in.Charge.ChargeBase,
			ServicePeriod:             rateableIntent.ServicePeriod,
			AmountAfterProration:      rateableIntent.AmountAfterProration,
			NoFiatTransactionRequired: ratingResult.Totals.Total.IsZero(),
			Immutable:                 false,
			LineID:                    lo.ToPtr(in.LineID),
			InvoiceID:                 lo.ToPtr(in.InvoiceID),
		})
		if err != nil {
			return StartCreditThenInvoiceRunResult{}, fmt.Errorf("create current run: %w", err)
		}

		result := StartCreditThenInvoiceRunResult{
			Run: flatfee.RealizationRun{
				RealizationRunBase: runBase,
			},
		}

		charge := in.Charge
		charge.Realizations.CurrentRun = &flatfee.RealizationRun{
			RealizationRunBase: runBase,
			DetailedLines:      mo.Some(ratingResult.DetailedLines),
		}

		creditAllocationTarget := currency.RoundToPrecision(ratingResult.Totals.Total)
		if !creditAllocationTarget.IsZero() {
			handlerInput := flatfee.OnAllocateCreditsInput{
				Charge:                 charge,
				ServicePeriod:          rateableIntent.ServicePeriod,
				BookedAt:               flatfee.UsageBookedAt(charge.Intent.GetEffectivePaymentTerm(), rateableIntent.ServicePeriod),
				PreTaxAmountToAllocate: creditAllocationTarget,
			}
			if err := handlerInput.Validate(); err != nil {
				return StartCreditThenInvoiceRunResult{}, fmt.Errorf("validating allocate credits input: %w", err)
			}

			creditAllocations, err := s.handler.OnAllocateCredits(ctx, handlerInput)
			if err != nil {
				return StartCreditThenInvoiceRunResult{}, fmt.Errorf("allocate credits for flat fee: %w", err)
			}

			creditAllocationsWithLineID := creditrealization.CreateAllocationInputs(lo.Map(creditAllocations, func(allocation creditrealization.CreateAllocationInput, _ int) creditrealization.CreateAllocationInput {
				allocation.LineID = lo.ToPtr(in.LineID)
				return allocation
			}))

			if len(creditAllocationsWithLineID) > 0 {
				realizations, err := s.createCreditAllocations(ctx, charge, runBase.ID, creditAllocationsWithLineID.AsCreateInputs())
				if err != nil {
					return StartCreditThenInvoiceRunResult{}, fmt.Errorf("creating credit realizations: %w", err)
				}

				result.Run.CreditRealizations = realizations
			}
		}

		allocated := currency.RoundToPrecision(result.Run.CreditRealizations.Sum())
		if allocated.GreaterThan(creditAllocationTarget) {
			return StartCreditThenInvoiceRunResult{}, fmt.Errorf(
				"credit allocations exceed rated total [charge_id=%s total=%s allocated=%s]",
				in.Charge.ID,
				creditAllocationTarget.String(),
				allocated.String(),
			)
		}

		creditsApplied, err := result.Run.CreditRealizations.AsCreditsApplied()
		if err != nil {
			return StartCreditThenInvoiceRunResult{}, fmt.Errorf("mapping credit realizations to credits applied: %w", err)
		}

		detailedLines, err := ratingResult.DetailedLines.WithCreditsApplied(creditsApplied, currency)
		if err != nil {
			return StartCreditThenInvoiceRunResult{}, fmt.Errorf("applying credits to detailed lines: %w", err)
		}

		if err := s.adapter.UpsertDetailedLines(ctx, runBase.ID, detailedLines); err != nil {
			return StartCreditThenInvoiceRunResult{}, fmt.Errorf("persisting detailed lines for run[%s]: %w", runBase.ID.ID, err)
		}

		runTotals := detailedLines.SumTotals().RoundToPrecision(currency)

		runBase, err = s.adapter.UpdateRealizationRun(ctx, flatfee.UpdateRealizationRunInput{
			ID:                        runBase.ID,
			Totals:                    mo.Some(runTotals),
			NoFiatTransactionRequired: mo.Some(runTotals.Total.IsZero()),
		})
		if err != nil {
			return StartCreditThenInvoiceRunResult{}, fmt.Errorf("updating run totals for run[%s]: %w", runBase.ID.ID, err)
		}

		result.Run.RealizationRunBase = runBase
		result.Run.DetailedLines = mo.Some(detailedLines)

		return result, nil
	})
}

type ReconcileStandardLineToIntentInput struct {
	Charge flatfee.Charge
	Run    flatfee.RealizationRun
	// AllocateAt is used as the ledger timestamp when reconciliation needs to
	// allocate or correct credit rows.
	AllocateAt time.Time
}

func (i ReconcileStandardLineToIntentInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if err := i.Run.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("run: %w", err))
	}

	if i.AllocateAt.IsZero() {
		errs = append(errs, errors.New("allocate at is required"))
	}

	if i.Run.LineID == nil || *i.Run.LineID == "" {
		errs = append(errs, errors.New("run line ID is required"))
	}

	if i.Run.InvoiceID == nil || *i.Run.InvoiceID == "" {
		errs = append(errs, errors.New("run invoice ID is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type ReconcileStandardLineToIntentResult struct {
	Run flatfee.RealizationRun
}

// ReconcileStandardLineToIntent rerates a mutable credit_then_invoice run from
// the effective charge intent and reconciles its credit allocations.
func (s *Service) ReconcileStandardLineToIntent(ctx context.Context, in ReconcileStandardLineToIntentInput) (ReconcileStandardLineToIntentResult, error) {
	if err := in.Validate(); err != nil {
		return ReconcileStandardLineToIntentResult{}, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (ReconcileStandardLineToIntentResult, error) {
		currency := in.Charge.Intent.GetCurrency()

		rateableIntent, err := in.Charge.GetRateableIntent()
		if err != nil {
			return ReconcileStandardLineToIntentResult{}, fmt.Errorf("getting rateable intent: %w", err)
		}

		ratingResult, err := s.Rate(rateableIntent)
		if err != nil {
			return ReconcileStandardLineToIntentResult{}, fmt.Errorf("rating flat fee: %w", err)
		}

		run := in.Run
		run.ServicePeriod = rateableIntent.ServicePeriod

		creditAllocationTarget := currency.RoundToPrecision(ratingResult.Totals.Total)
		reconcileResult, err := s.ReconcileCredits(ctx, ReconcileCreditRealizationsInput{
			Charge:             in.Charge,
			Run:                run,
			AllocateAt:         in.AllocateAt,
			TargetAmount:       creditAllocationTarget,
			CurrencyCalculator: currency,
		})
		if err != nil {
			return ReconcileStandardLineToIntentResult{}, fmt.Errorf("reconcile credits for run %s: %w", run.ID.ID, err)
		}

		run.CreditRealizations = append(run.CreditRealizations, reconcileResult.Realizations...)

		allocated := currency.RoundToPrecision(run.CreditRealizations.Sum())
		if allocated.GreaterThan(creditAllocationTarget) {
			return ReconcileStandardLineToIntentResult{}, fmt.Errorf(
				"credit allocations exceed rated total [charge_id=%s total=%s allocated=%s]",
				in.Charge.ID,
				creditAllocationTarget.String(),
				allocated.String(),
			)
		}

		creditsApplied, err := run.CreditRealizations.AsCreditsApplied()
		if err != nil {
			return ReconcileStandardLineToIntentResult{}, fmt.Errorf("mapping credit realizations to credits applied: %w", err)
		}

		detailedLines, err := ratingResult.DetailedLines.WithCreditsApplied(creditsApplied, currency)
		if err != nil {
			return ReconcileStandardLineToIntentResult{}, fmt.Errorf("applying credits to detailed lines: %w", err)
		}

		if err := s.adapter.UpsertDetailedLines(ctx, run.ID, detailedLines); err != nil {
			return ReconcileStandardLineToIntentResult{}, fmt.Errorf("persisting detailed lines for run[%s]: %w", run.ID.ID, err)
		}

		runTotals := detailedLines.SumTotals().RoundToPrecision(currency)

		runBase, err := s.adapter.UpdateRealizationRun(ctx, flatfee.UpdateRealizationRunInput{
			ID:                        run.ID,
			ServicePeriod:             mo.Some(rateableIntent.ServicePeriod),
			AmountAfterProration:      mo.Some(rateableIntent.AmountAfterProration),
			Totals:                    mo.Some(runTotals),
			NoFiatTransactionRequired: mo.Some(runTotals.Total.IsZero()),
		})
		if err != nil {
			return ReconcileStandardLineToIntentResult{}, fmt.Errorf("updating run totals for run[%s]: %w", run.ID.ID, err)
		}

		run.RealizationRunBase = runBase
		run.DetailedLines = mo.Some(detailedLines)

		return ReconcileStandardLineToIntentResult{Run: run}, nil
	})
}
