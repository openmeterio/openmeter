package timeutil

import "time"

type OpenPeriod struct {
	From *time.Time `json:"from,omitempty"`
	To   *time.Time `json:"to,omitempty"`
}

var _ Period = OpenPeriod{}

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
