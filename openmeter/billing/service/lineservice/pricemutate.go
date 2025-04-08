package lineservice

import "time"

type priceMutator struct {
	Pricer          Pricer
	PostCalculation []PostCalculationMutator
}

type PostCalculationMutator interface {
	Mutate(PricerCalculateInput, newDetailedLinesInput) (newDetailedLinesInput, error)
}

var _ Pricer = (*priceMutator)(nil)

func (p *priceMutator) Calculate(l PricerCalculateInput) (newDetailedLinesInput, error) {
	newDetailedLines, err := p.Pricer.Calculate(l)
	if err != nil {
		return nil, err
	}

	// Apply post-calculation mutators
	for _, mutator := range p.PostCalculation {
		newDetailedLines, err = mutator.Mutate(l, newDetailedLines)
		if err != nil {
			return nil, err
		}
	}

	return newDetailedLines, nil
}

func (p *priceMutator) CanBeInvoicedAsOf(l usageBasedLine, asOf time.Time) (bool, error) {
	return p.Pricer.CanBeInvoicedAsOf(l, asOf)
}
