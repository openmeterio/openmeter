package service

import (
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/billing/pricer"
	"github.com/openmeterio/openmeter/openmeter/billing/pricer/service/price"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

// ValidateStandardLine validates the standard line and returns an error if the line is invalid/inconsistent
func validateStandardLine(in pricer.StandardLineAccessor) error {
	price := in.GetPrice()
	if price == nil {
		return fmt.Errorf("price is nil")
	}

	progressivelyBilledServicePeriod, err := in.GetProgressivelyBilledServicePeriod()
	if err != nil {
		return fmt.Errorf("getting progressively billed service period: %w", err)
	}

	// Validate the progressive billing related information
	if !in.IsProgressivelyBilled() && !progressivelyBilledServicePeriod.Equal(in.GetServicePeriod()) {
		return fmt.Errorf("full service period does not match the service period for a non-progressively billed line")
	}

	return nil
}

func (s *service) GenerateDetailedLines(in pricer.StandardLineAccessor) (pricer.GenerateDetailedLinesResult, error) {
	if err := validateStandardLine(in); err != nil {
		return pricer.GenerateDetailedLinesResult{}, fmt.Errorf("validating billable line: %w", err)
	}

	currencyCalc, err := in.GetCurrency().Calculator()
	if err != nil {
		return pricer.GenerateDetailedLinesResult{}, fmt.Errorf("creating currency calculator: %w", err)
	}

	linePricer, err := getPricerFor(in)
	if err != nil {
		return pricer.GenerateDetailedLinesResult{}, fmt.Errorf("creating pricer: %w", err)
	}

	fullProgressivelyBilledServicePeriod, err := in.GetProgressivelyBilledServicePeriod()
	if err != nil {
		return pricer.GenerateDetailedLinesResult{}, fmt.Errorf("getting progressively billed service period: %w", err)
	}

	input := price.PricerCalculateInput{
		StandardLineAccessor:                 in,
		CurrencyCalculator:                   currencyCalc,
		FullProgressivelyBilledServicePeriod: fullProgressivelyBilledServicePeriod,
		StandardLineDiscounts:                in.GetStandardLineDiscounts(),
	}

	if in.GetPrice().Type() != productcatalog.FlatPriceType {
		meteredQuantity, err := in.GetMeteredQuantity()
		if err != nil {
			return pricer.GenerateDetailedLinesResult{}, fmt.Errorf("getting metered usage: %w", err)
		}

		preLinePeriodMeteredQuantity, err := in.GetMeteredPreLinePeriodQuantity()
		if err != nil {
			return pricer.GenerateDetailedLinesResult{}, fmt.Errorf("getting pre line period metered usage: %w", err)
		}

		input.Usage = &pricer.Usage{
			Quantity:              *meteredQuantity,
			PreLinePeriodQuantity: *preLinePeriodMeteredQuantity,
		}
	}

	if err := input.Validate(); err != nil {
		return pricer.GenerateDetailedLinesResult{}, fmt.Errorf("validating pricer input: %w", err)
	}

	out, err := linePricer.GenerateDetailedLines(input)
	if err != nil {
		return pricer.GenerateDetailedLinesResult{}, fmt.Errorf("calculating detailed lines: %w", err)
	}

	if out == nil {
		return pricer.GenerateDetailedLinesResult{}, fmt.Errorf("detailed lines are nil")
	}

	outWithTotals := getTotalsFromDetailedLines(*out, currencyCalc)

	return outWithTotals, nil
}

// UpdateTotalsFromDetailedLines is a helper method to update the totals of a line from its detailed lines.
func getTotalsFromDetailedLines(in pricer.GenerateDetailedLinesResult, calc currencyx.Calculator) pricer.GenerateDetailedLinesResult {
	// Calculate the line totals
	for idx, detailedLine := range in.DetailedLines {
		in.DetailedLines[idx].Totals = calculateDetailedLineTotals(detailedLine, calc)
	}

	// WARNING: Even if tempting to add discounts etc. here to the totals, we should always keep the logic as is.
	// The usageBasedLine will never be synchronized directly to stripe or other apps, only the detailed lines.
	//
	// Given that the external systems will have their own logic for calculating the totals, we cannot expect
	// any custom logic implemented here to be carried over to the external systems.

	// UBP line's value is the sum of all the children
	in.Totals = totals.Sum(
		lo.Map(in.DetailedLines, func(l pricer.DetailedLine, _ int) totals.Totals {
			return l.Totals
		})...,
	)

	return in
}

func calculateDetailedLineTotals(line pricer.DetailedLine, calc currencyx.Calculator) totals.Totals {
	// Calculate the line totals
	totals := totals.Totals{
		DiscountsTotal: line.AmountDiscounts.SumAmount(calc),
		CreditsTotal:   line.CreditsApplied.SumAmount(calc),

		// TODO[OM-979]: implement taxes
		TaxesInclusiveTotal: alpacadecimal.Zero,
		TaxesExclusiveTotal: alpacadecimal.Zero,
		TaxesTotal:          alpacadecimal.Zero,
	}

	amount := calc.RoundToPrecision(line.PerUnitAmount.Mul(line.Quantity))

	switch line.Category {
	case billing.FlatFeeCategoryCommitment:
		totals.ChargesTotal = amount
	default:
		totals.Amount = amount
	}

	totals.Total = totals.CalculateTotal()

	return totals
}
