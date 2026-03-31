package service

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

type getRatingForUsageResult struct {
	rating.GenerateDetailedLinesResult
	Quantity alpacadecimal.Decimal
}

type getRatingForUsageInput struct {
	Charge         usagebased.Charge
	Customer       billing.CustomerOverrideWithDetails
	FeatureMeter   feature.FeatureMeter
	StoredAtOffset time.Time
}

func (i getRatingForUsageInput) Validate() error {
	if err := i.Charge.Validate(); err != nil {
		return fmt.Errorf("charge: %w", err)
	}

	if i.Customer.Customer == nil {
		return fmt.Errorf("customer is required")
	}

	if i.FeatureMeter.Meter == nil {
		return fmt.Errorf("feature meter is required")
	}

	if i.StoredAtOffset.IsZero() {
		return fmt.Errorf("stored at offset is required")
	}

	return nil
}

func (s *service) getRatingForUsage(ctx context.Context, in getRatingForUsageInput) (getRatingForUsageResult, error) {
	if err := in.Validate(); err != nil {
		return getRatingForUsageResult{}, err
	}

	currencyCalculator, err := in.Charge.Intent.Currency.Calculator()
	if err != nil {
		return getRatingForUsageResult{}, fmt.Errorf("creating currency calculator: %w", err)
	}

	snapshotQuantity, err := s.snapshotQuantity(ctx, snapshotQuantityInput{
		Customer:       in.Customer.Customer,
		FeatureMeter:   in.FeatureMeter,
		ServicePeriod:  in.Charge.Intent.ServicePeriod,
		StoredAtOffset: in.StoredAtOffset,
	})
	if err != nil {
		return getRatingForUsageResult{}, fmt.Errorf("get snapshot quantity: %w", err)
	}

	ratingResult, err := s.ratingService.GenerateDetailedLines(usagebased.RateableIntent{
		Intent:     in.Charge.Intent,
		MeterValue: snapshotQuantity,
	})
	if err != nil {
		return getRatingForUsageResult{}, fmt.Errorf("rating: %w", err)
	}

	return getRatingForUsageResult{
		GenerateDetailedLinesResult: roundGenerateDetailedLinesResultTotals(ratingResult, currencyCalculator),
		Quantity:                    snapshotQuantity,
	}, nil
}

func roundGenerateDetailedLinesResultTotals(in rating.GenerateDetailedLinesResult, calc currencyx.Calculator) rating.GenerateDetailedLinesResult {
	in.Totals = roundTotalsToCurrencyPrecision(in.Totals, calc)

	return in
}

func roundTotalsToCurrencyPrecision(in totals.Totals, calc currencyx.Calculator) totals.Totals {
	in.Amount = calc.RoundToPrecision(in.Amount)
	in.ChargesTotal = calc.RoundToPrecision(in.ChargesTotal)
	in.DiscountsTotal = calc.RoundToPrecision(in.DiscountsTotal)
	in.TaxesInclusiveTotal = calc.RoundToPrecision(in.TaxesInclusiveTotal)
	in.TaxesExclusiveTotal = calc.RoundToPrecision(in.TaxesExclusiveTotal)
	in.TaxesTotal = calc.RoundToPrecision(in.TaxesTotal)
	in.CreditsTotal = calc.RoundToPrecision(in.CreditsTotal)
	in.Total = calc.RoundToPrecision(in.Total)

	return in
}
