package streaming

import (
	"context"
	"errors"
	"time"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/meterevent"
	"github.com/openmeterio/openmeter/openmeter/namespace"
	"github.com/openmeterio/openmeter/pkg/models"
)

type CountEventsParams struct {
	From time.Time
}

// CountEventRow represents a row in the count events response.
type CountEventRow struct {
	Count   uint64
	Subject string
}

// RawEvent represents a single raw event
type RawEvent struct {
	Namespace  string    `ch:"namespace" json:"-"`
	ID         string    `ch:"id" json:"id"`
	Type       string    `ch:"type" json:"type"`
	Source     string    `ch:"source" json:"source"`
	Subject    string    `ch:"subject" json:"subject"`
	Time       time.Time `ch:"time" json:"time"`
	Data       string    `ch:"data" json:"data"`
	IngestedAt time.Time `ch:"ingested_at" json:"ingested_at,omitempty,omitzero"`
	StoredAt   time.Time `ch:"stored_at" json:"stored_at,omitempty,omitzero"`
	StoreRowID string    `ch:"store_row_id" json:"store_row_id,omitempty,omitzero"`
}

type Connector interface {
	namespace.Handler

	CountEvents(ctx context.Context, namespace string, params CountEventsParams) ([]CountEventRow, error)
	ListEvents(ctx context.Context, namespace string, params meterevent.ListEventsParams) ([]RawEvent, error)
	ListEventsV2(ctx context.Context, params meterevent.ListEventsV2Params) ([]RawEvent, error)
	QueryMeter(ctx context.Context, namespace string, meter meter.Meter, params QueryParams) ([]meter.MeterQueryRow, error)
	ListMeterSubjects(ctx context.Context, namespace string, meter meter.Meter, params ListMeterSubjectsParams) ([]string, error)
	BatchInsert(ctx context.Context, events []RawEvent) error
	ValidateJSONPath(ctx context.Context, jsonPath string) (bool, error)
}

// ListMeterSubjectsParams is a parameter object for listing subjects.
type ListMeterSubjectsParams struct {
	From *time.Time
	To   *time.Time
}

// Validate validates the list meters parameters.
func (p ListMeterSubjectsParams) Validate() error {
	var errs []error

	if p.From != nil && p.To != nil {
		if p.From.Equal(*p.To) {
			errs = append(errs, errors.New("from and to cannot be equal"))
		}

		if p.From.After(*p.To) {
			errs = append(errs, errors.New("from time must be before to time"))
		}
	}

	if len(errs) > 0 {
		return models.NewNillableGenericValidationError(errors.Join(errs...))
	}

	return errors.Join(errs...)
}
