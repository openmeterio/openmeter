package lineservice

import "github.com/openmeterio/openmeter/pkg/timeutil"

type priceMutator struct {
	PreCalculation  []PreCalculationMutator
	Pricer          Pricer
	PostCalculation []PostCalculationMutator
}

type PostCalculationMutator interface {
	Mutate(PricerCalculateInput, newDetailedLinesInput) (newDetailedLinesInput, error)
}

type PreCalculationMutator interface {
	Mutate(PricerCalculateInput) (PricerCalculateInput, error)
}

var _ Pricer = (*priceMutator)(nil)

func (p *priceMutator) Calculate(l PricerCalculateInput) (newDetailedLinesInput, error) {
	// Apply pre-calculation mutators
	for _, mutator := range p.PreCalculation {
		var err error

		l, err = mutator.Mutate(l)
		if err != nil {
			return nil, err
		}
	}

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

func (p *priceMutator) CanBeInvoicedAsOf(in CanBeInvoicedAsOfInput) (*timeutil.ClosedPeriod, error) {
	return p.Pricer.CanBeInvoicedAsOf(in)
}
