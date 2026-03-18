package service

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/pricer"
	"github.com/openmeterio/openmeter/openmeter/billing/pricer/service/mutator"
	"github.com/openmeterio/openmeter/openmeter/billing/pricer/service/price"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func getPricerFor(line pricer.PriceAccessor) (*priceMutator, error) {
	linePrice := line.GetPrice()
	if linePrice == nil {
		return nil, errors.New("price is nil")
	}

	if linePrice.Type() == productcatalog.FlatPriceType {
		return &priceMutator{
			Pricer: price.Flat{},
			PostCalculation: []mutator.PostCalculationMutator{
				&mutator.DiscountPercentage{},
				&mutator.Credits{},
			},
		}, nil
	}

	var basePricer price.Pricer

	switch linePrice.Type() {
	case productcatalog.UnitPriceType:
		basePricer = price.Unit{}
	case productcatalog.TieredPriceType:
		basePricer = price.Tiered{}
	case productcatalog.PackagePriceType:
		basePricer = price.Package{}
	case productcatalog.DynamicPriceType:
		basePricer = price.Dynamic{}
	default:
		return nil, fmt.Errorf("unsupported price type: %s", linePrice.Type())
	}

	// This priceMutator captures the calculation flow for discounts and commitments:
	return &priceMutator{
		PreCalculation: []mutator.PreCalculationMutator{
			&mutator.DiscountUsage{},
		},
		Pricer: basePricer,
		PostCalculation: []mutator.PostCalculationMutator{
			&mutator.DiscountPercentage{},
			&mutator.MaxAmountCommitment{},
			&mutator.MinAmountCommitment{},
			&mutator.Credits{},
		},
	}, nil
}

type priceMutator struct {
	PreCalculation  []mutator.PreCalculationMutator
	Pricer          price.Pricer
	PostCalculation []mutator.PostCalculationMutator
}

func (p *priceMutator) GenerateDetailedLines(l price.PricerCalculateInput) (*pricer.GenerateDetailedLinesResult, error) {
	mutatedInput := l
	// Apply pre-calculation mutators
	for _, mutator := range p.PreCalculation {
		var err error

		mutatedInput, err = mutator.Mutate(mutatedInput)
		if err != nil {
			return nil, err
		}
	}

	detailedLines, err := p.Pricer.GenerateDetailedLines(mutatedInput)
	if err != nil {
		return nil, err
	}

	// Apply post-calculation mutators
	for _, mutator := range p.PostCalculation {
		detailedLines, err = mutator.Mutate(mutatedInput, detailedLines)
		if err != nil {
			return nil, err
		}
	}

	return &pricer.GenerateDetailedLinesResult{
		DetailedLines:              detailedLines,
		FinalUsage:                 mutatedInput.Usage,
		FinalStandardLineDiscounts: mutatedInput.StandardLineDiscounts,
	}, nil
}

func (p *priceMutator) ResolveBillablePeriod(in pricer.ResolveBillablePeriodInput) (*timeutil.ClosedPeriod, error) {
	return p.Pricer.ResolveBillablePeriod(in)
}
