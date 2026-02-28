package sync

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/llmcost"
)

// Fetcher retrieves raw price data from an external source.
type Fetcher interface {
	// Source returns the identifier for this data source.
	Source() llmcost.PriceSource

	// Fetch retrieves all current prices from the source.
	Fetch(ctx context.Context) ([]llmcost.SourcePrice, error)
}
