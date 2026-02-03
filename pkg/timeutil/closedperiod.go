package timeutil

import (
	"errors"
	"time"
)

type ClosedPeriod struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

var _ Period = ClosedPeriod{}

func (p ClosedPeriod) Duration() time.Duration {
	return p.To.Sub(p.From)
}

// Inclusive at both start and end
func (p ClosedPeriod) ContainsInclusive(t time.Time) bool {
	if t.Equal(p.From) || t.Equal(p.To) {
		return true
	}

	return t.After(p.From) && t.Before(p.To)
}

// Exclusive at both start and end
func (p ClosedPeriod) ContainsExclusive(t time.Time) bool {
	return p.Contains(t) && !t.Equal(p.From)
}

// Inclusive at start, exclusive at end
func (p ClosedPeriod) Contains(t time.Time) bool {
	return (t.After(p.From) || t.Equal(p.From)) && t.Before(p.To)
}

// Returns true if the two periods overlap at any point
// Returns false if the periods are exactly sequential, e.g.: [1, 2] and [2, 3]
func (p ClosedPeriod) Overlaps(other ClosedPeriod) bool {
	// If one period ends before or exactly when the other starts, they don't overlap
	switch {
	case p.To.Before(other.From) || p.To.Equal(other.From):
		return false
	case other.To.Before(p.From) || other.To.Equal(p.From):
		return false
	default:
		return true
	}
}

// Returns true if the two periods overlap at any point
// Returns true if the periods are exactly sequential, e.g.: [1, 2] and [2, 3]
func (p ClosedPeriod) OverlapsInclusive(other ClosedPeriod) bool {
	return p.ContainsInclusive(other.From) || p.ContainsInclusive(other.To) || other.ContainsInclusive(p.From) || other.ContainsInclusive(p.To)
}

func (p ClosedPeriod) Intersection(other ClosedPeriod) *ClosedPeriod {
	// Calculate the latest From date (intersection starts at the later of the two start times)
	var newFrom time.Time
	if p.From.After(other.From) {
		newFrom = p.From
	} else {
		newFrom = other.From
	}

	// Calculate the earliest To date (intersection ends at the earlier of the two end times)
	var newTo time.Time
	if p.To.Before(other.To) {
		newTo = p.To
	} else {
		newTo = other.To
	}

	// Check if the periods overlap
	// If the start is at or after the end, there's no overlap
	if !newFrom.Before(newTo) {
		return nil
	}

	return &ClosedPeriod{
		From: newFrom,
		To:   newTo,
	}
}

func (p ClosedPeriod) Open() OpenPeriod {
	return OpenPeriod{
		From: &p.From,
		To:   &p.To,
	}
}

func (p ClosedPeriod) Validate() error {
	if p.From.After(p.To) {
		return errors.New("from must be before to")
	}

	return nil
}

func (p ClosedPeriod) Truncate(resolution time.Duration) ClosedPeriod {
	return ClosedPeriod{
		From: p.From.Truncate(resolution),
		To:   p.To.Truncate(resolution),
	}
}

func (p ClosedPeriod) Equals(other ClosedPeriod) bool {
	return p.From.Equal(other.From) && p.To.Equal(other.To)
}
