package rating

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	billingrating "github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

type DetailedLinesFetcher interface {
	FetchDetailedLines(ctx context.Context, charge usagebased.Charge) (usagebased.Charge, error)
}

type Config struct {
	StreamingConnector   streaming.Connector
	RatingService        billingrating.Service
	DetailedLinesFetcher DetailedLinesFetcher
}

func (c Config) Validate() error {
	var errs []error

	if c.StreamingConnector == nil {
		errs = append(errs, errors.New("streaming connector cannot be null"))
	}

	if c.RatingService == nil {
		errs = append(errs, errors.New("rating service cannot be null"))
	}

	if c.DetailedLinesFetcher == nil {
		errs = append(errs, errors.New("detailed lines fetcher cannot be null"))
	}

	return errors.Join(errs...)
}

type Service interface {
	// GetTotalsForUsage returns charge totals for a usage snapshot without generating detailed lines.
	// Prefer this when only totals are required because it is faster than generating detailed lines.
	GetTotalsForUsage(ctx context.Context, in GetTotalsForUsageInput) (totals.Totals, error)
	// GetDetailedRatingForUsage returns rated detailed lines and the metered quantity snapshot used to compute them.
	// Prefer GetTotalsForUsage when only totals are required because it is faster.
	GetDetailedRatingForUsage(ctx context.Context, in GetDetailedRatingForUsageInput) (GetDetailedRatingForUsageResult, error)
	// GetPreferredRatingEngineFor returns the preferred rating engine for a given intent.
	GetPreferredRatingEngineFor(intent usagebased.Intent) usagebased.RatingEngine
}

func (s *service) GetPreferredRatingEngineFor(_ usagebased.Intent) usagebased.RatingEngine {
	return usagebased.RatingEngineDelta
}

type service struct {
	streamingConnector   streaming.Connector
	ratingService        billingrating.Service
	detailedLinesFetcher DetailedLinesFetcher
}

func New(config Config) (Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &service{
		streamingConnector:   config.StreamingConnector,
		ratingService:        config.RatingService,
		detailedLinesFetcher: config.DetailedLinesFetcher,
	}, nil
}
