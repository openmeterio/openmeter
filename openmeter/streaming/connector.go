package streaming

import (
	"context"
	"errors"
	"time"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/namespace"
	"github.com/openmeterio/openmeter/pkg/models"
)

type ListEventsParams struct {
	ClientID       *string
	From           time.Time
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

// RawEvent represents a single raw event
type RawEvent struct {
	Namespace  string
	ID         string
	Type       string
	Source     string
	Subject    string
	Time       time.Time
	Data       string
	IngestedAt time.Time
	StoredAt   time.Time
}

type Connector interface {
	namespace.Handler

	CountEvents(ctx context.Context, namespace string, params CountEventsParams) ([]CountEventRow, error)
	ListEvents(ctx context.Context, namespace string, params ListEventsParams) ([]api.IngestedEvent, error)
	CreateMeter(ctx context.Context, namespace string, meter meter.Meter) error
	UpdateMeter(ctx context.Context, namespace string, meter meter.Meter) error
	DeleteMeter(ctx context.Context, namespace string, meter meter.Meter) error
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
