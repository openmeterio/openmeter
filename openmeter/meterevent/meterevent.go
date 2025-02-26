package meterevent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/api"
)

const (
	MaximumFromDuration     = time.Hour * 24 * 32 // 32 days
	MaximumLimit        int = 100
)

type Service interface {
	ListEvents(ctx context.Context, input ListEventsInput) ([]api.IngestedEvent, error)
}

// ListEventsInput represents the input for ListEvents method.
type ListEventsInput struct {
	// The namespace.
	Namespace string
	// Start date-time. Inclusive.
	IngestedAtFrom *time.Time
	// End date-time. Inclusive.
	IngestedAtTo *time.Time
	// If not provided lists all events.
	// If provided with true, only list events with processing error.
	// If provided with false, only list events without processing error. */
	HasError *bool
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

// Validate validates the input.
func (i ListEventsInput) Validate() error {
	var errs []error

	minimumFrom := time.Now().Add(-MaximumFromDuration)

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
