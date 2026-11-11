package rte

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)

// UpstreamResolver maps source facts to tracked calculations. Resolution never
// calculates balances or creates demand for customers that have no interest.
type UpstreamResolver interface {
	Resolve(context.Context, marshaler.Event) ([]DemandSelector, error)
}
