package timeutil

import (
	"fmt"
	"time"
)

type OpenPeriod struct {
	From *time.Time `json:"from,omitempty"`
	To   *time.Time `json:"to,omitempty"`
}

var _ Period = OpenPeriod{}

func (p OpenPeriod) Equals(other OpenPeriod) bool {
	// For From: both should be nil, or both non-nil and equal
	if (p.From == nil) != (other.From == nil) {
		return false
	}
	if p.From != nil && !p.From.Equal(*other.From) {
		return false
	}

	// For To: both should be nil, or both non-nil and equal
	if (p.To == nil) != (other.To == nil) {
		return false
	}
	if p.To != nil && !p.To.Equal(*other.To) {
		return false
	}

	return true
}

// Inclusive at both start and end
func (p OpenPeriod) ContainsInclusive(t time.Time) bool {
	if p.From != nil && t.Before(*p.From) {
		return false
	}

	if p.To != nil && t.After(*p.To) {
		return false
	}

	return true
}

// Exclusive at both start and end
func (p OpenPeriod) ContainsExclusive(t time.Time) bool {
	if p.From != nil && (t.Before(*p.From) || t.Equal(*p.From)) {
		return false
	}

	if p.To != nil && (t.After(*p.To) || t.Equal(*p.To)) {
		return false
	}

	return true
}

// Inclusive at start, exclusive at end
func (p OpenPeriod) Contains(t time.Time) bool {
	if p.From != nil && t.Before(*p.From) {
		return false
	}

	if p.To != nil && (t.After(*p.To) || t.Equal(*p.To)) {
		return false
	}

	return true
}

func (p OpenPeriod) Intersection(other OpenPeriod) *OpenPeriod {
	// If either period is completely empty, return a copy of the other
	if p.From == nil && p.To == nil && other.From == nil && other.To == nil {
		return &OpenPeriod{}
	}

	// Calculate the latest From date
	var newFrom *time.Time
	switch {
	case p.From == nil && other.From == nil:
		newFrom = nil
	case p.From == nil:
		tmp := *other.From
		newFrom = &tmp
	case other.From == nil:
		tmp := *p.From
		newFrom = &tmp
	default:
		// Both From are not nil, take the later one
		if p.From.After(*other.From) {
			tmp := *p.From
			newFrom = &tmp
		} else {
			tmp := *other.From
			newFrom = &tmp
		}
	}

	// Calculate the earliest To date
	var newTo *time.Time
	switch {
	case p.To == nil && other.To == nil:
		newTo = nil
	case p.To == nil:
		tmp := *other.To
		newTo = &tmp
	case other.To == nil:
		tmp := *p.To
		newTo = &tmp
	default:
		// Both To are not nil, take the earlier one
		if p.To.Before(*other.To) {
			tmp := *p.To
			newTo = &tmp
		} else {
			tmp := *other.To
			newTo = &tmp
		}
	}

	// Check if the periods overlap
	if newFrom != nil && newTo != nil {
		// If the start is at or after the end, there's no overlap
		if !newFrom.Before(*newTo) {
			return nil
		}
	}

	// Check the case where periods are open-ended in opposite directions
	if (p.From != nil && other.To != nil && !p.From.Before(*other.To)) ||
		(other.From != nil && p.To != nil && !other.From.Before(*p.To)) {
		return nil
	}

	return &OpenPeriod{
		From: newFrom,
		To:   newTo,
	}
}

// Difference returns P - Other (times in P not in Other)
func (p OpenPeriod) Difference(other OpenPeriod) []OpenPeriod {
	// Check for intersection
	intersection := p.Intersection(other)

	// If there's no intersection, the difference is the original period
	if intersection == nil {
		return []OpenPeriod{p}
	}

	// If the intersection equals the original period, the difference is empty
	if (p.From == nil) == (intersection.From == nil) &&
		(p.To == nil) == (intersection.To == nil) &&
		(p.From == nil || p.From.Equal(*intersection.From)) &&
		(p.To == nil || p.To.Equal(*intersection.To)) {
		return []OpenPeriod{}
	}

	result := []OpenPeriod{}

	// Check if there's a period before the intersection
	if (p.From == nil && intersection.From != nil) ||
		(p.From != nil && intersection.From != nil && p.From.Before(*intersection.From)) {
		before := OpenPeriod{
			From: p.From,
			To:   intersection.From,
		}
		result = append(result, before)
	}

	// Check if there's a period after the intersection
	if (p.To == nil && intersection.To != nil) ||
		(p.To != nil && intersection.To != nil && p.To.After(*intersection.To)) {
		after := OpenPeriod{
			From: intersection.To,
			To:   p.To,
		}
		result = append(result, after)
	}

	return result
}

func (p OpenPeriod) Union(other OpenPeriod) OpenPeriod {
	// If either period is empty, return an empty period
	if (p.From == nil && p.To == nil) || (other.From == nil && other.To == nil) {
		return OpenPeriod{}
	}

	// Calculate the earliest From date
	var newFrom *time.Time
	switch {
	case p.From == nil || other.From == nil:
		newFrom = nil
	case p.From.Before(*other.From):
		tmp := *p.From
		newFrom = &tmp
	default:
		tmp := *other.From
		newFrom = &tmp
	}

	// Calculate the latest To date
	var newTo *time.Time
	switch {
	case p.To == nil || other.To == nil:
		newTo = nil
	case p.To.After(*other.To):
		tmp := *p.To
		newTo = &tmp
	default:
		tmp := *other.To
		newTo = &tmp
	}

	return OpenPeriod{
		From: newFrom,
		To:   newTo,
	}
}

// IsSupersetOf returns true if p contains other (both ends inclusive)
func (p OpenPeriod) IsSupersetOf(other OpenPeriod) bool {
	// Empty period is a superset of everything
	if p.From == nil && p.To == nil {
		return true
	}

	// Non-empty period is not a superset of empty period
	if other.From == nil && other.To == nil {
		return false
	}

	// Check From boundary
	if p.From != nil {
		// If p has a From but other doesn't, p is not a superset
		if other.From == nil {
			return false
		}
		// If p starts after other, p is not a superset
		if p.From.After(*other.From) {
			return false
		}
	}

	// Check To boundary
	if p.To != nil {
		// If p has a To but other doesn't, p is not a superset
		if other.To == nil {
			return false
		}
		// If p ends before other, p is not a superset
		if p.To.Before(*other.To) {
			return false
		}
	}

	// Check for touching periods - they are not considered supersets
	if (p.To != nil && other.From != nil && p.To.Equal(*other.From)) ||
		(p.From != nil && other.To != nil && p.From.Equal(*other.To)) {
		return false
	}

	return true
}

// Returns true if the two periods overlap at any point.
// Also returns true if the periods are exactly sequential, e.g.: [1, 2] and [2, 3]
func (p OpenPeriod) OverlapsInclusive(other OpenPeriod) bool {
	// If they have an intersection, they overlap
	if p.Intersection(other) != nil {
		return true
	}

	// If they are sequential, they overlap
	if (p.From != nil && other.To != nil && p.From.Equal(*other.To)) ||
		(p.To != nil && other.From != nil && p.To.Equal(*other.From)) {
		return true
	}

	// Otherwise, they don't overlap
	return false
}

func (p OpenPeriod) Closed() (ClosedPeriod, error) {
	if p.From == nil || p.To == nil {
		return ClosedPeriod{}, fmt.Errorf("cannot convert open period to closed period with nil boundaries")
	}

	return ClosedPeriod{
		From: *p.From,
		To:   *p.To,
	}, nil
}
