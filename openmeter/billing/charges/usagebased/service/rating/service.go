package usagebasedrating

import (
	"context"

	usagebasedrating "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/streaming"
)

type Service interface {
	GetRatingForUsage(ctx context.Context, input GetRatingForUsageInput) (GetRatingForUsageResult, error)
}

type service struct {
	streamingConnector streaming.Connector
	ratingService      rating.Service
}

func New() usagebasedrating.Service {
	return &service{}
}
