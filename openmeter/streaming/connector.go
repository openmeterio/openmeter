package streaming

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/pkg/models"
)

type CountEventsParams struct {
	From time.Time
}

// CountEventRow represents a row in the count events response.
type CountEventRow struct {
	Count   uint64
	Subject string
	IsError bool
}

type Connector interface {
	CountEvents(ctx context.Context, namespace string, params CountEventsParams) ([]CountEventRow, error)
	ListEvents(ctx context.Context, namespace string, params ListEventsParams) ([]api.IngestedEvent, *EventsCursor, error)
	PaginateEvents(ctx context.Context, namespace string, params PaginateEventsParams) ([]api.IngestedEvent, *EventsCursor, error)
	CreateMeter(ctx context.Context, namespace string, meter *models.Meter) error
	DeleteMeter(ctx context.Context, namespace string, meterSlug string) error
	QueryMeter(ctx context.Context, namespace string, meterSlug string, params *QueryParams) ([]models.MeterQueryRow, error)
	ListMeterSubjects(ctx context.Context, namespace string, meterSlug string, from *time.Time, to *time.Time) ([]string, error)
	// Add more methods as needed ...
}
