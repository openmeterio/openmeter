package streaming

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/namespace"
)

type ListEventsParams struct {
	From           *time.Time
	To             *time.Time
	IngestedAtFrom *time.Time
	IngestedAtTo   *time.Time
	ID             *string
	Subject        *string
	HasError       *bool
	Limit          int
}

type CountEventsParams struct {
	From time.Time
}

// CountEventRow represents a row in the count events response.
type CountEventRow struct {
	Count   uint64
	Subject string
	IsError bool
}

type ListMeterSubjectsParams struct {
	From *time.Time
	To   *time.Time
}

// RawEvent represents a single raw event
type RawEvent struct {
	Namespace       string
	ValidationError string
	ID              string
	Type            string
	Source          string
	Subject         string
	Time            time.Time
	Data            string
	IngestedAt      time.Time
	StoredAt        time.Time
}

type Connector interface {
	namespace.Handler

	CountEvents(ctx context.Context, namespace string, params CountEventsParams) ([]CountEventRow, error)
	ListEvents(ctx context.Context, namespace string, params ListEventsParams) ([]api.IngestedEvent, error)
	CreateMeter(ctx context.Context, namespace string, meter models.Meter) error
	DeleteMeter(ctx context.Context, namespace string, meter models.Meter) error
	QueryMeter(ctx context.Context, namespace string, meter models.Meter, params QueryParams) ([]models.MeterQueryRow, error)
	ListMeterSubjects(ctx context.Context, namespace string, meter models.Meter, params ListMeterSubjectsParams) ([]string, error)
	BatchInsert(ctx context.Context, events []RawEvent) error
	// Add more methods as needed ...
}
