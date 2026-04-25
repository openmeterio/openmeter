package streaming

import (
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination/v2"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

// EventSortField is the column used to order events returned by ListEventsV2.
type EventSortField string

const (
	EventSortFieldTime       EventSortField = "time"
	EventSortFieldIngestedAt EventSortField = "ingested_at"
	EventSortFieldStoredAt   EventSortField = "stored_at"
)

// Values returns the known values for the event sort field.
func (f EventSortField) Values() []string {
	return []string{
		string(EventSortFieldTime),
		string(EventSortFieldIngestedAt),
		string(EventSortFieldStoredAt),
	}
}

// Validate returns an error when the sort field is not one of the known values.
// The zero value is treated as valid and resolved to EventSortFieldTime at query time.
func (f EventSortField) Validate() error {
	if f == "" {
		return nil
	}

	if !lo.Contains(f.Values(), string(f)) {
		return models.NewGenericValidationError(fmt.Errorf("invalid event sort value: %s", f))
	}

	return nil
}

// ListEventsParams represents the input for ListEvents method.
type ListEventsParams struct {
	// The namespace.
	Namespace string
	// The client ID.
	ClientID *string
	// Start date-time. Inclusive.
	IngestedAtFrom *time.Time
	// End date-time. Inclusive.
	IngestedAtTo *time.Time
	// The event ID. Accepts partial ID.
	ID *string
	// The event subject. Accepts partial subject.
	Subject *string
	// The event customer ID.
	Customers *[]Customer
	// Start date-time. Inclusive.
	From time.Time
	// End date-time. Inclusive.
	To *time.Time
	// Number of events to return.
	Limit int
}

// Validate validates the input.
func (i ListEventsParams) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.ClientID != nil && *i.ClientID == "" {
		errs = append(errs, errors.New("client id cannot be empty"))
	}

	if i.From.IsZero() {
		errs = append(errs, errors.New("from date is required"))
	}

	if i.To != nil && i.To.Before(i.From) {
		errs = append(errs, fmt.Errorf("to date is before from date: %s < %s", i.To.Format(time.RFC3339), i.From.Format(time.RFC3339)))
	}

	if i.IngestedAtFrom != nil && i.IngestedAtTo != nil && i.IngestedAtTo.Before(*i.IngestedAtFrom) {
		errs = append(errs, fmt.Errorf("ingestedAtTo date is before ingestedAtFrom date: %s < %s", i.IngestedAtTo.Format(time.RFC3339), i.IngestedAtFrom.Format(time.RFC3339)))
	}

	return errors.Join(errs...)
}

// ListEventsV2Params is a parameter object for listing events.
type ListEventsV2Params struct {
	// The namespace.
	Namespace string
	// The client ID.
	ClientID *string
	// The cursor.
	Cursor *pagination.Cursor
	// The limit.
	Limit *int
	// The ID filter.
	ID *filter.FilterString
	// The source filter.
	Source *filter.FilterString
	// The subject filter.
	Subject *filter.FilterString
	// The event customer ID.
	Customers *[]Customer
	// The type filter.
	Type *filter.FilterString
	// The time filter.
	Time *filter.FilterTime
	// The ingested at filter.
	IngestedAt *filter.FilterTime
	// The stored at filter.
	StoredAt *filter.FilterTime
	// The sort column; zero value defaults to EventSortFieldTime.
	SortBy EventSortField
	// The sort direction; zero value defaults to sortx.OrderDesc.
	SortOrder sortx.Order
}

// Validate validates the list events parameters.
func (p ListEventsV2Params) Validate() error {
	var errs []error

	if p.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if p.Cursor != nil {
		if err := p.Cursor.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("cursor: %w", err))
		}
	}

	if p.ID != nil {
		if err := p.ID.ValidateWithComplexity(1); err != nil {
			errs = append(errs, fmt.Errorf("id: %w", err))
		}
	}

	if p.Source != nil {
		if err := p.Source.ValidateWithComplexity(1); err != nil {
			errs = append(errs, fmt.Errorf("source: %w", err))
		}
	}

	if p.Subject != nil {
		if err := p.Subject.ValidateWithComplexity(1); err != nil {
			errs = append(errs, fmt.Errorf("subject: %w", err))
		}
	}

	if p.Type != nil {
		if err := p.Type.ValidateWithComplexity(1); err != nil {
			errs = append(errs, fmt.Errorf("type: %w", err))
		}
	}

	if p.Time != nil {
		if err := p.Time.ValidateWithComplexity(1); err != nil {
			errs = append(errs, fmt.Errorf("time: %w", err))
		}
	}

	if p.IngestedAt != nil {
		if err := p.IngestedAt.ValidateWithComplexity(1); err != nil {
			errs = append(errs, fmt.Errorf("ingested_at: %w", err))
		}
	}

	if p.StoredAt != nil {
		if err := p.StoredAt.ValidateWithComplexity(1); err != nil {
			errs = append(errs, fmt.Errorf("stored_at: %w", err))
		}
	}

	if err := p.SortBy.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("sort by: %w", err))
	}

	if p.Limit != nil && *p.Limit < 1 {
		errs = append(errs, errors.New("limit must be greater than 0"))
	}

	return errors.Join(errs...)
}
