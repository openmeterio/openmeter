package rating

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	billingrating "github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

type Config struct {
	StreamingConnector streaming.Connector
	RatingService      billingrating.Service
}

func (c Config) Validate() error {
	var errs []error

	if c.StreamingConnector == nil {
		errs = append(errs, errors.New("streaming connector cannot be null"))
	}

	if c.RatingService == nil {
		errs = append(errs, errors.New("rating service cannot be null"))
	}

	return errors.Join(errs...)
}

type Service struct {
	streamingConnector streaming.Connector
	ratingService      billingrating.Service
}

func New(config Config) (*Service, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Service{
		streamingConnector: config.StreamingConnector,
		ratingService:      config.RatingService,
	}, nil
}

type GetRatingForUsageResult struct {
	billingrating.GenerateDetailedLinesResult
	Quantity alpacadecimal.Decimal
}

type GetRatingForUsageInput struct {
	Charge         usagebased.Charge
	Customer       billing.CustomerOverrideWithDetails
	FeatureMeter   feature.FeatureMeter
	StoredAtOffset time.Time
}

func (i GetRatingForUsageInput) Validate() error {
	if err := i.Charge.Validate(); err != nil {
		return fmt.Errorf("charge: %w", err)
	}

	if i.Customer.Customer == nil {
		return fmt.Errorf("customer is required")
	}

	if i.FeatureMeter.Meter == nil {
		return fmt.Errorf("feature meter is required")
	}

	if i.StoredAtOffset.IsZero() {
		return fmt.Errorf("stored at offset is required")
	}

	return nil
}

func (s *Service) GetRatingForUsage(ctx context.Context, in GetRatingForUsageInput) (GetRatingForUsageResult, error) {
	if err := in.Validate(); err != nil {
		return GetRatingForUsageResult{}, err
	}

	snapshotQuantity, err := s.snapshotQuantity(ctx, snapshotQuantityInput{
		Customer:       in.Customer.Customer,
		FeatureMeter:   in.FeatureMeter,
		ServicePeriod:  in.Charge.Intent.ServicePeriod,
		StoredAtOffset: in.StoredAtOffset,
	})
	if err != nil {
		return GetRatingForUsageResult{}, fmt.Errorf("get snapshot quantity: %w", err)
	}

	ratingResult, err := s.ratingService.GenerateDetailedLines(usagebased.RateableIntent{
		Intent:     in.Charge.Intent,
		MeterValue: snapshotQuantity,
	})
	if err != nil {
		return GetRatingForUsageResult{}, fmt.Errorf("rating: %w", err)
	}

	return GetRatingForUsageResult{
		GenerateDetailedLinesResult: ratingResult,
		Quantity:                    snapshotQuantity,
	}, nil
}
