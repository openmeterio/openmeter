package meterevent

import (
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/pagination/v2"
)

// Subject represents a subject that has ingested events.
type Subject struct {
	// The subject key.
	Key string
}

var _ pagination.Item = (*Subject)(nil)

// Cursor returns the cursor for the subject. Subjects are ordered by key, so the
// cursor carries the key in its ID part; the time part is unused.
func (s Subject) Cursor() pagination.Cursor {
	return pagination.NewCursor(time.Time{}, s.Key)
}

// ListSubjectsParams is a parameter object for listing subjects.
type ListSubjectsParams struct {
	// The namespace.
	Namespace string
	// The cursor.
	Cursor *pagination.Cursor
	// The limit.
	Limit *int
	// The subject key filter.
	Key *filter.FilterString
	// Attributed filters subjects by whether they are attributed to a customer.
	Attributed *bool
}

// Validate validates the list subjects parameters.
func (p ListSubjectsParams) Validate() error {
	var errs []error

	if p.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if p.Cursor != nil && p.Cursor.ID == "" {
		errs = append(errs, errors.New("cursor id is required"))
	}

	if p.Key != nil {
		if err := p.Key.ValidateWithComplexity(1); err != nil {
			errs = append(errs, fmt.Errorf("key: %w", err))
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
