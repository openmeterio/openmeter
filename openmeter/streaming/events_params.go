package streaming

import (
	"fmt"
	"time"
)

type EventsTableFilters struct {
	From           *time.Time
	To             *time.Time
	IngestedAtFrom *time.Time
	IngestedAtTo   *time.Time
	ID             *string
	Subject        *string
	HasError       *bool
}

// EventsTableCursor contains all properties by which the events table is ordered deterministically.
type EventsTableCursor struct {
	Namespace string
	Time      time.Time
	Type      string
	Subject   string
	ID        string

	IsGreater bool
}

func (c EventsTableCursor) Validate() error {
	if c.Namespace == "" {
		return fmt.Errorf("All properties are required")
	}
	if c.Time.IsZero() {
		return fmt.Errorf("All properties are required")
	}
	if c.Type == "" {
		return fmt.Errorf("All properties are required")
	}
	if c.Subject == "" {
		return fmt.Errorf("All properties are required")
	}
	if c.ID == "" {
		return fmt.Errorf("All properties are required")
	}

	return nil
}

type EventsCursor struct {
	Cursor  EventsTableCursor
	Filters EventsTableFilters
}

type ListEventsParams struct {
	Filters EventsTableFilters
	Limit   int
}

type PaginateEventsParams struct {
	Cursor EventsCursor
	Limit  int
}
