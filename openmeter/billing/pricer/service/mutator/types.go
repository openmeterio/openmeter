package mutator

import (
	"github.com/openmeterio/openmeter/openmeter/billing/pricer"
	"github.com/openmeterio/openmeter/openmeter/billing/pricer/service/price"
)

type PostCalculationMutator interface {
	Mutate(price.PricerCalculateInput, pricer.DetailedLines) (pricer.DetailedLines, error)
}

type PreCalculationMutator interface {
	Mutate(price.PricerCalculateInput) (price.PricerCalculateInput, error)
}
