package streaming

import (
	"context"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"

	"github.com/openmeterio/openmeter/pkg/models"
)

type ListEventsParams struct {
	Limit int
}

type QueryParams struct {
	From           *time.Time
	To             *time.Time
	Subject        []string
	GroupBySubject bool
	GroupBy        []string
	Aggregation    models.MeterAggregation
	WindowSize     *models.WindowSize
}

type QueryResult struct {
	WindowSize *models.WindowSize
	Values     []*models.MeterValue
}

type Connector interface {
	ListEvents(ctx context.Context, namespace string, params ListEventsParams) ([]event.Event, error)
	CreateMeter(ctx context.Context, namespace string, meter *models.Meter) error
	DeleteMeter(ctx context.Context, namespace string, meterSlug string) error
	QueryMeter(ctx context.Context, namespace string, meterSlug string, params *QueryParams) (*QueryResult, error)
	ListMeterSubjects(ctx context.Context, namespace string, meterSlug string, from *time.Time, to *time.Time) ([]string, error)
	// Add more methods as needed ...
}
