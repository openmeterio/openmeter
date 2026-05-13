package realizations

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/invoiceupdater"
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

	if i.Line.ChargeID == nil || *i.Line.ChargeID != i.Charge.ID {
		errs = append(errs, fmt.Errorf("line charge id must match charge"))
	}

	if i.Line.InvoiceID != i.Invoice.ID {
		errs = append(errs, fmt.Errorf("line invoice id must match invoice"))
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
			handlerInput := flatfee.OnAssignedToInvoiceInput{
				Charge:            charge,
				ServicePeriod:     in.Line.Period,
				PreTaxTotalAmount: amountAfterProration,
			}
			if err := handlerInput.Validate(); err != nil {
				return StartCreditThenInvoiceRunResult{}, fmt.Errorf("validating on assigned to invoice input: %w", err)
			}

			creditAllocations, err := s.handler.OnAssignedToInvoice(ctx, handlerInput)
			if err != nil {
				return StartCreditThenInvoiceRunResult{}, fmt.Errorf("on flat fee assigned to invoice: %w", err)
			}

			for idx := range creditAllocations {
				creditAllocations[idx].LineID = lo.ToPtr(in.Line.ID)
			}

			if len(creditAllocations) > 0 {
				realizations, err := s.createCreditAllocations(ctx, charge, runBase.ID, creditAllocations.AsCreateInputs())
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
