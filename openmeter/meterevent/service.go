package meterevent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/pagination/v2"
)

const (
	MaximumFromDuration     = time.Hour * 24 * 32 // 32 days
	MaximumLimit        int = 100
)

type Service interface {
	ListEvents(ctx context.Context, params ListEventsParams) ([]Event, error)
	ListEventsV2(ctx context.Context, params ListEventsV2Params) (pagination.Result[Event], error)
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
	CustomerIDs *[]string
	// Start date-time. Inclusive.
	From time.Time
	// End date-time. Inclusive.
	To *time.Time
	// Number of events to return.
	Limit int
}

// Event represents a single event.
type Event struct {
	// The event ID.
	ID string
	// The event type.
	Type string
	// The event source.
	Source string
	// The event subject.
	Subject string
	// The event time.
	Time time.Time
	// The event data as a JSON string.
	Data string
	// The event customer ID.
	CustomerID *string
	// The time the event was ingested.
	IngestedAt time.Time
	// The time the event was stored.
	StoredAt time.Time
	// Validation errors.
	ValidationErrors []error
}

var _ pagination.Item = (*Event)(nil)

// Cursor returns the cursor for the event.
func (e Event) Cursor() pagination.Cursor {
	return pagination.NewCursor(e.Time, e.ID)
}

// Validate validates the input.
func (i ListEventsParams) Validate() error {
	var errs []error

	minimumFrom := time.Now().Add(-MaximumFromDuration)

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.ClientID != nil && *i.ClientID == "" {
		errs = append(errs, errors.New("client id cannot be empty"))
	}

	if i.From.IsZero() {
		errs = append(errs, errors.New("from date is required"))
	}

	if minimumFrom.After(i.From) {
		errs = append(errs, fmt.Errorf("from date is too old: %s, must be after %s", i.From.Format(time.RFC3339), minimumFrom.Format(time.RFC3339)))
	}

	if i.To != nil && i.To.Before(i.From) {
		errs = append(errs, fmt.Errorf("to date is before from date: %s < %s", i.To.Format(time.RFC3339), i.From.Format(time.RFC3339)))
	}

	if i.IngestedAtFrom != nil && minimumFrom.After(*i.IngestedAtFrom) {
		errs = append(errs, fmt.Errorf("ingestedAtFrom date is too old, must be after: %s: %s", i.IngestedAtFrom.Format(time.RFC3339), minimumFrom.Format(time.RFC3339)))
	}

	if i.IngestedAtFrom != nil && i.IngestedAtTo != nil && i.IngestedAtTo.Before(*i.IngestedAtFrom) {
		errs = append(errs, fmt.Errorf("ingestedAtTo date is before ingestedAtFrom date: %s < %s", i.IngestedAtTo.Format(time.RFC3339), i.IngestedAtFrom.Format(time.RFC3339)))
	}

	if i.Limit < 1 {
		errs = append(errs, errors.New("limit must be greater than 0"))
	}

	if i.Limit > MaximumLimit {
		errs = append(errs, fmt.Errorf("limit must be less than or equal to %d", MaximumLimit))
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
	// The customer ID filter.
	CustomerID *filter.FilterString
	// The type filter.
	Type *filter.FilterString
	// The time filter.
	Time *filter.FilterTime
	// The ingested at filter.
	IngestedAt *filter.FilterTime
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

	if p.CustomerID != nil {
		if err := p.CustomerID.ValidateWithComplexity(1); err != nil {
			errs = append(errs, fmt.Errorf("customer id: %w", err))
		}

		// Only $in is supported for customer id
		if !p.CustomerID.IsEmpty() && p.CustomerID.In == nil {
			errs = append(errs, errors.New("customer id filter supports only in"))
		}
	}

	if p.Type != nil {
		if err := p.Type.ValidateWithComplexity(1); err != nil {
			errs = append(errs, fmt.Errorf("type: %w", err))
		}
	}

	if p.Time != nil && p.IngestedAt != nil && !p.Time.IsEmpty() && !p.IngestedAt.IsEmpty() {
		errs = append(errs, errors.New("time and ingested_at cannot both be set"))
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

	if p.Limit != nil && *p.Limit < 1 {
		errs = append(errs, errors.New("limit must be greater than 0"))
	}

	if p.Limit != nil && *p.Limit > MaximumLimit {
		errs = append(errs, fmt.Errorf("limit must be less than or equal to %d", MaximumLimit))
	}

	return errors.Join(errs...)
}
