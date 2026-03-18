package mutator

import (
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/billing/rating/service/rate"
)

type PostCalculationMutator interface {
	Mutate(rate.PricerCalculateInput, rating.DetailedLines) (rating.DetailedLines, error)
}

type PreCalculationMutator interface {
	Mutate(rate.PricerCalculateInput) (rate.PricerCalculateInput, error)
}
