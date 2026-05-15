package realizations

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/service/invoicecalc"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

type StartCreditThenInvoiceRunInput struct {
	Charge  flatfee.Charge
	Line    billing.StandardLine
	Invoice billing.StandardInvoice
}

func (i StartCreditThenInvoiceRunInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if err := i.Line.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("line: %w", err))
	}

	if err := i.Invoice.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invoice: %w", err))
	}

	lineChargeID := "<nil>"
	if i.Line.ChargeID != nil {
		lineChargeID = *i.Line.ChargeID
	}

	if i.Line.ChargeID == nil || *i.Line.ChargeID != i.Charge.ID {
		errs = append(errs, fmt.Errorf("line charge id mismatch: got %s, want %s", lineChargeID, i.Charge.ID))
	}

	if i.Line.InvoiceID != i.Invoice.ID {
		errs = append(errs, fmt.Errorf("line invoice id mismatch: got %s, want %s", i.Line.InvoiceID, i.Invoice.ID))
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
		currencyCalculator, err := in.Charge.Intent.Currency.Calculator()
		if err != nil {
			return StartCreditThenInvoiceRunResult{}, fmt.Errorf("get currency calculator: %w", err)
		}

		amountAfterProration, err := invoiceupdater.GetFlatFeePerUnitAmount(&in.Line)
		if err != nil {
			return StartCreditThenInvoiceRunResult{}, fmt.Errorf("get flat fee line amount: %w", err)
		}

		amountAfterProration = currencyCalculator.RoundToPrecision(amountAfterProration)

		runBase, err := s.adapter.CreateCurrentRun(ctx, flatfee.CreateCurrentRunInput{
			Charge:                    in.Charge.ChargeBase,
			ServicePeriod:             in.Line.Period,
			AmountAfterProration:      amountAfterProration,
			NoFiatTransactionRequired: amountAfterProration.IsZero(),
			Immutable:                 false,
			LineID:                    lo.ToPtr(in.Line.ID),
			InvoiceID:                 lo.ToPtr(in.Invoice.ID),
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
		}

		if !amountAfterProration.IsZero() {
			handlerInput := flatfee.OnAllocateCreditsInput{
				Charge:                 charge,
				ServicePeriod:          in.Line.Period,
				PreTaxAmountToAllocate: amountAfterProration,
			}
			if err := handlerInput.Validate(); err != nil {
				return StartCreditThenInvoiceRunResult{}, fmt.Errorf("validating allocate credits input: %w", err)
			}

			creditAllocations, err := s.handler.OnAllocateCredits(ctx, handlerInput)
			if err != nil {
				return StartCreditThenInvoiceRunResult{}, fmt.Errorf("allocate credits for flat fee: %w", err)
			}

			creditAllocationsWithLineID := creditrealization.CreateAllocationInputs(lo.Map(creditAllocations, func(allocation creditrealization.CreateAllocationInput, _ int) creditrealization.CreateAllocationInput {
				allocation.LineID = lo.ToPtr(in.Line.ID)
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

		creditsApplied, err := result.Run.CreditRealizations.AsCreditsApplied()
		if err != nil {
			return StartCreditThenInvoiceRunResult{}, fmt.Errorf("mapping credit realizations to credits applied: %w", err)
		}

		line, err := in.Line.Clone()
		if err != nil {
			return StartCreditThenInvoiceRunResult{}, fmt.Errorf("cloning line: %w", err)
		}

		line.CreditsApplied = creditsApplied

		generatedDetailedLines, err := s.ratingService.GenerateDetailedLines(line)
		if err != nil {
			return StartCreditThenInvoiceRunResult{}, fmt.Errorf("generating detailed lines for line[%s]: %w", line.ID, err)
		}

		if err := invoicecalc.MergeGeneratedDetailedLines(line, generatedDetailedLines); err != nil {
			return StartCreditThenInvoiceRunResult{}, fmt.Errorf("merging generated detailed lines for line[%s]: %w", line.ID, err)
		}

		if err := line.Validate(); err != nil {
			return StartCreditThenInvoiceRunResult{}, fmt.Errorf("validating standard line[%s]: %w", line.ID, err)
		}

		detailedLines := flatfee.DetailedLines(lo.Map(line.DetailedLines, func(detailedLine billing.DetailedLine, _ int) flatfee.DetailedLine {
			return detailedLine.Base.Clone()
		}))

		if err := s.adapter.UpsertDetailedLines(ctx, runBase.ID, detailedLines); err != nil {
			return StartCreditThenInvoiceRunResult{}, fmt.Errorf("persisting detailed lines for line[%s]: %w", line.ID, err)
		}

		runBase, err = s.adapter.UpdateRealizationRun(ctx, flatfee.UpdateRealizationRunInput{
			ID:                        runBase.ID,
			Totals:                    mo.Some(line.Totals),
			NoFiatTransactionRequired: mo.Some(line.Totals.Total.IsZero()),
		})
		if err != nil {
			return StartCreditThenInvoiceRunResult{}, fmt.Errorf("updating run totals for line[%s]: %w", line.ID, err)
		}

		result.Run.RealizationRunBase = runBase
		result.Run.DetailedLines = mo.Some(detailedLines)

		return result, nil
	})
}

// ReconcileStandardLineToIntentInput describes a mutable CTI standard invoice
// line that has already been rebuilt from the latest charge intent, plus the
// realization run that still reflects the previous line state.
type ReconcileStandardLineToIntentInput struct {
	// Charge is the flat-fee charge whose intent produced Line.
	Charge flatfee.Charge
	// Run is the current mutable realization run backing Line.
	Run flatfee.RealizationRun
	// Line is the desired standard invoice line after applying the latest
	// charge intent.
	Line billing.StandardLine
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

	if err := i.Line.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("line: %w", err))
	}

	if i.AllocateAt.IsZero() {
		errs = append(errs, errors.New("allocate at is required"))
	}

	lineChargeID := lo.FromPtrOr(i.Line.ChargeID, "<nil>")
	if lineChargeID != i.Charge.ID {
		errs = append(errs, fmt.Errorf("line charge id mismatch: got %s, want %s", lineChargeID, i.Charge.ID))
	}

	runLineID := lo.FromPtrOr(i.Run.LineID, "<nil>")

	if runLineID != i.Line.ID {
		errs = append(errs, fmt.Errorf("run line id mismatch: got %s, want %s", runLineID, i.Line.ID))
	}

	runInvoiceID := lo.FromPtrOr(i.Run.InvoiceID, "<nil>")

	if runInvoiceID != i.Line.InvoiceID {
		errs = append(errs, fmt.Errorf("run invoice id mismatch: got %s, want %s", runInvoiceID, i.Line.InvoiceID))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// ReconcileStandardLineToIntentResult returns both sides of the reconciliation:
// the persisted run aggregate and the standard line that billing should write back.
type ReconcileStandardLineToIntentResult struct {
	Run flatfee.RealizationRun
	// Line includes recalculated credits, detailed lines, and totals.
	Line billing.StandardLine
}

// ReconcileStandardLineToIntent brings a mutable credit_then_invoice standard
// invoice line and its realization run back in sync after the charge intent
// changed.
//
// The caller passes the freshly rebuilt standard line. This method treats that
// line as the desired state, computes its prorated amount, reconciles the run's
// credit allocations to that amount, maps the resulting credit realizations
// back to billing CreditsApplied, regenerates detailed lines/totals, persists
// charge-owned detailed lines, and updates the run aggregate.
func (s *Service) ReconcileStandardLineToIntent(ctx context.Context, in ReconcileStandardLineToIntentInput) (ReconcileStandardLineToIntentResult, error) {
	if err := in.Validate(); err != nil {
		return ReconcileStandardLineToIntentResult{}, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (ReconcileStandardLineToIntentResult, error) {
		currencyCalculator, err := in.Charge.Intent.Currency.Calculator()
		if err != nil {
			return ReconcileStandardLineToIntentResult{}, fmt.Errorf("get currency calculator: %w", err)
		}

		amountAfterProration, err := invoiceupdater.GetFlatFeePerUnitAmount(&in.Line)
		if err != nil {
			return ReconcileStandardLineToIntentResult{}, fmt.Errorf("get flat fee line amount: %w", err)
		}

		amountAfterProration = currencyCalculator.RoundToPrecision(amountAfterProration)

		run := in.Run
		// The rebuilt line may carry a prorated period that differs from the
		// persisted run. Use the line period for both credit allocation and the
		// run update so ledger and invoice state describe the same service
		// window.
		run.ServicePeriod = in.Line.Period
		reconcileResult, err := s.ReconcileCredits(ctx, ReconcileCreditRealizationsInput{
			Charge:             in.Charge,
			Run:                run,
			AllocateAt:         in.AllocateAt,
			TargetAmount:       amountAfterProration,
			CurrencyCalculator: currencyCalculator,
		})
		if err != nil {
			return ReconcileStandardLineToIntentResult{}, fmt.Errorf("reconcile credits for run %s: %w", run.ID.ID, err)
		}

		run.CreditRealizations = append(run.CreditRealizations, reconcileResult.Realizations...)

		creditsApplied, err := run.CreditRealizations.AsCreditsApplied()
		if err != nil {
			return ReconcileStandardLineToIntentResult{}, fmt.Errorf("mapping credit realizations to credits applied: %w", err)
		}

		line, err := in.Line.Clone()
		if err != nil {
			return ReconcileStandardLineToIntentResult{}, fmt.Errorf("cloning line: %w", err)
		}

		// CreditsApplied is invoice-facing derived data. The durable credit
		// source of truth remains the run's credit realization rows.
		line.CreditsApplied = creditsApplied

		generatedDetailedLines, err := s.ratingService.GenerateDetailedLines(line)
		if err != nil {
			return ReconcileStandardLineToIntentResult{}, fmt.Errorf("generating detailed lines for line[%s]: %w", line.ID, err)
		}

		if err := invoicecalc.MergeGeneratedDetailedLines(line, generatedDetailedLines); err != nil {
			return ReconcileStandardLineToIntentResult{}, fmt.Errorf("merging generated detailed lines for line[%s]: %w", line.ID, err)
		}

		if err := line.Validate(); err != nil {
			return ReconcileStandardLineToIntentResult{}, fmt.Errorf("validating standard line[%s]: %w", line.ID, err)
		}

		detailedLines := flatfee.DetailedLines(lo.Map(line.DetailedLines, func(detailedLine billing.DetailedLine, _ int) flatfee.DetailedLine {
			return detailedLine.Base.Clone()
		}))

		if err := s.adapter.UpsertDetailedLines(ctx, run.ID, detailedLines); err != nil {
			return ReconcileStandardLineToIntentResult{}, fmt.Errorf("persisting detailed lines for line[%s]: %w", line.ID, err)
		}

		runBase, err := s.adapter.UpdateRealizationRun(ctx, flatfee.UpdateRealizationRunInput{
			ID:                        run.ID,
			ServicePeriod:             mo.Some(line.Period),
			AmountAfterProration:      mo.Some(amountAfterProration),
			Totals:                    mo.Some(line.Totals),
			NoFiatTransactionRequired: mo.Some(line.Totals.Total.IsZero()),
		})
		if err != nil {
			return ReconcileStandardLineToIntentResult{}, fmt.Errorf("updating run totals for line[%s]: %w", line.ID, err)
		}

		run.RealizationRunBase = runBase
		run.DetailedLines = mo.Some(detailedLines)

		return ReconcileStandardLineToIntentResult{
			Run:  run,
			Line: *line,
		}, nil
	})
}
