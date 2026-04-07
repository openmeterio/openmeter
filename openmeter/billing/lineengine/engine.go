package lineengine

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
)

var _ billing.LineEngine = (*Engine)(nil)

type Config struct {
	SplitLineGroupAdapter SplitLineGroupAdapter
	QuantitySnapshotter   QuantitySnapshotter
	RatingService         rating.Service
}

func (c Config) Validate() error {
	if c.SplitLineGroupAdapter == nil {
		return fmt.Errorf("split line group adapter is required")
	}

	if c.QuantitySnapshotter == nil {
		return fmt.Errorf("quantity snapshotter is required")
	}

	if c.RatingService == nil {
		return fmt.Errorf("rating service is required")
	}

	return nil
}

type Engine struct {
	adapter             SplitLineGroupAdapter
	quantitySnapshotter QuantitySnapshotter
	ratingService       rating.Service
}

func New(config Config) (*Engine, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Engine{
		adapter:             config.SplitLineGroupAdapter,
		quantitySnapshotter: config.QuantitySnapshotter,
		ratingService:       config.RatingService,
	}, nil
}

func (e *Engine) GetLineEngineType() billing.LineEngineType {
	return billing.LineEngineTypeInvoice
}
