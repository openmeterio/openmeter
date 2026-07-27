package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/pkg/models"
)

type populateFlatFeeStandardLineFromRunInput struct {
	Charge flatfee.Charge
	Run    flatfee.RealizationRun
}

func (i populateFlatFeeStandardLineFromRunInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if err := i.Run.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("run: %w", err))
	}

	effectiveServicePeriod := i.Charge.Intent.GetEffectiveServicePeriod()
	if !i.Run.ServicePeriod.Equal(effectiveServicePeriod) {
		errs = append(errs, fmt.Errorf(
			"run service period does not match effective intent [run_id=%s run_period=%v intent_period=%v]",
			i.Run.ID.ID,
			i.Run.ServicePeriod,
			effectiveServicePeriod,
		))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func populateFlatFeeStandardLineFromRun(stdLine *billing.StandardLine, input populateFlatFeeStandardLineFromRunInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	if stdLine.UsageBased == nil {
		return fmt.Errorf("standard line[%s]: usage based line is required", stdLine.ID)
	}

	rateableIntent, err := input.Charge.GetRateableIntent()
	if err != nil {
		return fmt.Errorf("getting rateable intent: %w", err)
	}

	stdLine.Name = rateableIntent.GetName()
	stdLine.Period = rateableIntent.GetServicePeriod()
	stdLine.UsageBased.Price = rateableIntent.GetPrice()
	stdLine.RateCardDiscounts = rateableIntent.GetRateCardDiscounts()

	currency, err := stdLine.Currency.AsFiatCurrency()
	if err != nil {
		return fmt.Errorf("creating currency calculator: %w", err)
	}

	creditsApplied, err := input.Run.CreditRealizations.AsCreditsApplied()
	if err != nil {
		return err
	}

	stdLine.CreditsApplied = creditsApplied

	mappedDetailedLines, err := mapFlatFeeDetailedLines(stdLine, input.Run)
	if err != nil {
		return fmt.Errorf("mapping run detailed lines: %w", err)
	}

	mappedDetailedLines, err = mappedDetailedLines.WithCreditsApplied(stdLine.CreditsApplied, currency)
	if err != nil {
		return fmt.Errorf("applying run credits to detailed lines: %w", err)
	}

	stdLine.DetailedLines = stdLine.DetailedLinesWithIDReuse(mappedDetailedLines)
	stdLine.Totals = stdLine.DetailedLines.SumTotals().RoundToPrecision(currency)

	expectedTotals := input.Run.Totals.RoundToPrecision(currency)
	if !stdLine.Totals.Equal(expectedTotals) {
		return fmt.Errorf("mapped line totals do not match run totals [line_id=%s run_id=%s line_total=%s run_total=%s]",
			stdLine.ID, input.Run.ID.ID, stdLine.Totals.Total.String(), expectedTotals.Total.String())
	}

	return nil
}

func mapFlatFeeDetailedLines(stdLine *billing.StandardLine, run flatfee.RealizationRun) (billing.DetailedLines, error) {
	if run.DetailedLines.IsAbsent() {
		return nil, fmt.Errorf("run %s detailed lines must be expanded", run.ID.ID)
	}

	return lo.Map(run.DetailedLines.OrEmpty(), func(line flatfee.DetailedLine, _ int) billing.DetailedLine {
		base := line.Clone()
		base.Namespace = stdLine.Namespace
		base.ID = ""
		base.CreatedAt = time.Time{}
		base.UpdatedAt = time.Time{}
		base.DeletedAt = nil

		return billing.DetailedLine{
			DetailedLineBase: billing.DetailedLineBase{
				Base:      base,
				InvoiceID: stdLine.InvoiceID,
			},
		}
	}), nil
}
