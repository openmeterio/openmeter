package service

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/billing/rating/service/mutator"
	"github.com/openmeterio/openmeter/openmeter/billing/rating/service/rate"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func getPricerFor(line rating.PriceAccessor, opts rating.GenerateDetailedLinesOptions, unitConfigEnabled bool) (*priceMutator, error) {
	if line == nil {
		return nil, errors.New("line is nil")
	}

	if line.GetPrice() == nil {
		return nil, errors.New("price is nil")
	}

	linePrice := line.GetPrice()
	if linePrice == nil {
		return nil, errors.New("price is nil")
	}

	if linePrice.Type() == productcatalog.FlatPriceType {
		postCalculationMutators := []mutator.PostCalculationMutator{
			&mutator.DiscountPercentage{},
		}

		if !opts.DisableCreditsMutator {
			postCalculationMutators = append(postCalculationMutators, &mutator.Credits{})
		}

		return &priceMutator{
			Pricer:          rate.Flat{},
			PostCalculation: postCalculationMutators,
		}, nil
	}

	var basePricer rate.Pricer

	switch linePrice.Type() {
	case productcatalog.UnitPriceType:
		basePricer = rate.Unit{}
	case productcatalog.TieredPriceType:
		basePricer = rate.Tiered{}
	case productcatalog.PackagePriceType:
		basePricer = rate.Package{}
	case productcatalog.DynamicPriceType:
		basePricer = rate.Dynamic{}
	default:
		return nil, fmt.Errorf("unsupported price type: %s", linePrice.Type())
	}

	postCalculationMutators := []mutator.PostCalculationMutator{
		&mutator.DiscountPercentage{},
		&mutator.MaxAmountCommitment{},
	}

	// Charges pricing needs control over the minimum commitment as it should only be included for any
	// calculation that is after the service period end.
	if !opts.IgnoreMinimumCommitment {
		postCalculationMutators = append(postCalculationMutators, &mutator.MinAmountCommitment{})
	}

	if !opts.DisableCreditsMutator {
		postCalculationMutators = append(postCalculationMutators, &mutator.Credits{})
	}

	// UnitConfig converts the raw metered quantity into billed units and must run
	// before DiscountUsage so the usage discount applies to the converted, rounded
	// quantity (convert → round → discount). When the feature is disabled,
	// ForbidUnitConfig takes its place and errors if a line unexpectedly carries a
	// unit_config, so a dropped conversion surfaces instead of silently billing raw.
	preCalculationMutators := make([]mutator.PreCalculationMutator, 0, 2)
	if unitConfigEnabled {
		preCalculationMutators = append(preCalculationMutators, &mutator.UnitConfig{})
	} else {
		preCalculationMutators = append(preCalculationMutators, &mutator.ForbidUnitConfig{})
	}
	preCalculationMutators = append(preCalculationMutators, &mutator.DiscountUsage{})

	// This priceMutator captures the calculation flow for discounts and commitments:
	return &priceMutator{
		PreCalculation:  preCalculationMutators,
		Pricer:          basePricer,
		PostCalculation: postCalculationMutators,
	}, nil
}

type priceMutator struct {
	PreCalculation  []mutator.PreCalculationMutator
	Pricer          rate.Pricer
	PostCalculation []mutator.PostCalculationMutator
}

func (p *priceMutator) GenerateDetailedLines(l rate.PricerCalculateInput) (rating.GenerateDetailedLinesResult, error) {
	mutatedInput := l
	// Apply pre-calculation mutators
	for _, mutator := range p.PreCalculation {
		var err error

		mutatedInput, err = mutator.Mutate(mutatedInput)
		if err != nil {
			return rating.GenerateDetailedLinesResult{}, err
		}
	}

	detailedLines, err := p.Pricer.GenerateDetailedLines(mutatedInput)
	if err != nil {
		return rating.GenerateDetailedLinesResult{}, err
	}

	// Apply post-calculation mutators
	for _, mutator := range p.PostCalculation {
		detailedLines, err = mutator.Mutate(mutatedInput, detailedLines)
		if err != nil {
			return rating.GenerateDetailedLinesResult{}, err
		}
	}

	return rating.GenerateDetailedLinesResult{
		DetailedLines:              detailedLines,
		FinalUsage:                 mutatedInput.Usage,
		FinalStandardLineDiscounts: mutatedInput.StandardLineDiscounts,
	}, nil
}

func (p *priceMutator) ResolveBillablePeriod(in rating.ResolveBillablePeriodInput) (*timeutil.ClosedPeriod, error) {
	return p.Pricer.ResolveBillablePeriod(in)
}
