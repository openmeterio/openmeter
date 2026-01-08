package streaming

import (
	"context"
	"errors"
	"time"

	"github.com/openmeterio/openmeter/openmeter/meter"
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
	CustomerID *string   `ch:"customer_id" json:"customer_id,omitempty,omitzero"`
}

type Connector interface {
	namespace.Handler
	TableEngineRegistry

	CountEvents(ctx context.Context, namespace string, params CountEventsParams) ([]CountEventRow, error)
	ListEvents(ctx context.Context, namespace string, params ListEventsParams) ([]RawEvent, error)
	ListEventsV2(ctx context.Context, params ListEventsV2Params) ([]RawEvent, error)
	// ListSubjects lists the subjects that have events in the database
	ListSubjects(ctx context.Context, params ListSubjectsParams) ([]string, error)
	// ListGroupByValues lists the group by values that have events in the database
	ListGroupByValues(ctx context.Context, params ListGroupByValuesParams) ([]string, error)
	QueryMeter(ctx context.Context, namespace string, meter meter.Meter, params QueryParams) ([]meter.MeterQueryRow, error)
	BatchInsert(ctx context.Context, events []RawEvent) error
	ValidateJSONPath(ctx context.Context, jsonPath string) (bool, error)
}

// ListSubjectsParams is a parameter object for listing subjects.
type ListSubjectsParams struct {
	Namespace string
	Meter     *meter.Meter
	From      *time.Time
	To        *time.Time
	Search    *string
}

// Validate validates the list meters parameters.
func (p ListSubjectsParams) Validate() error {
	var errs []error

	if p.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if p.Meter != nil {
		if p.Meter.Key == "" {
			errs = append(errs, errors.New("meter cannot be empty when provided"))
		}
	}

	if p.From != nil && p.To != nil {
		if p.From.Equal(*p.To) {
			errs = append(errs, errors.New("from and to cannot be equal"))
		}

		if p.From.After(*p.To) {
			errs = append(errs, errors.New("from time must be before to time"))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// ListGroupByValuesParams is a parameter object for listing group by values.
type ListGroupByValuesParams struct {
	Namespace  string
	Meter      meter.Meter
	GroupByKey string
	From       *time.Time
	To         *time.Time
	Search     *string
}

// Validate validates the list group by values parameters.
func (p ListGroupByValuesParams) Validate() error {
	var errs []error

	if p.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if p.GroupByKey == "" {
		errs = append(errs, errors.New("group by key is required"))
	}

	if p.Meter.GroupBy[p.GroupByKey] == "" {
		errs = append(errs, errors.New("group by key is not valid for this meter"))
	}

	if p.From != nil {
		if time.Since(*p.From) >= time.Hour*24*90 {
			errs = append(errs, errors.New("from time must not be more than 90 days ago"))
		}
	}

	if p.From != nil && p.To != nil {
		if p.From.Equal(*p.To) {
			errs = append(errs, errors.New("from and to cannot be equal"))
		}

		if p.From.After(*p.To) {
			errs = append(errs, errors.New("from time must be before to time"))
		}

		if p.To.Sub(*p.From) > time.Hour*24*30 {
			errs = append(errs, errors.New("time window must be less than 30 days"))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
