package streaming

import (
	"context"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"

	"github.com/openmeterio/openmeter/pkg/models"
)

type QueryEventsParams struct {
	Limit int
}

type QueryParams struct {
	From           *time.Time
	To             *time.Time
	Subject        []string
	GroupBySubject bool
	GroupBy        []string
	Aggregation    *models.MeterAggregation
	WindowSize     *models.WindowSize
}

type QueryResult struct {
	WindowSize *models.WindowSize
	Values     []*models.MeterValue
}

type Connector interface {
	QueryEvents(ctx context.Context, namespace string, params QueryEventsParams) ([]event.Event, error)
	CreateMeter(ctx context.Context, namespace string, meter *models.Meter) error
	DeleteMeter(ctx context.Context, namespace string, meterSlug string) error
	QueryMeter(ctx context.Context, namespace string, meterSlug string, params *QueryParams) (*QueryResult, error)
	ListMeterSubjects(ctx context.Context, namespace string, meterSlug string) ([]string, error)
	// Add more methods as needed ...
}
