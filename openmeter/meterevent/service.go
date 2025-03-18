package meterevent

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const (
	MaximumFromDuration     = time.Hour * 24 * 32 // 32 days
	MaximumLimit        int = 100
)

type Service interface {
	ListEvents(ctx context.Context, params ListEventsParams) ([]Event, error)
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
	// The time the event was ingested.
	IngestedAt time.Time
	// The time the event was stored.
	StoredAt time.Time
	// Validation errors.
	ValidationErrors []error
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
