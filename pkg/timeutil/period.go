package timeutil

import (
	"time"
)

type Period struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

func (p Period) Duration() time.Duration {
	return p.To.Sub(p.From)
}

// Inclusive at both start and end
func (p Period) ContainsInclusive(t time.Time) bool {
	if t.Equal(p.From) || t.Equal(p.To) {
		return true
	}

	return t.After(p.From) && t.Before(p.To)
}

// Exclusive at both start and end
func (p Period) ContainsExclusive(t time.Time) bool {
	return p.Contains(t) && !t.Equal(p.From)
}

// Inclusive at start, exclusive at end
func (p Period) Contains(t time.Time) bool {
	return (t.After(p.From) || t.Equal(p.From)) && t.Before(p.To)
}

// Returns true if the two periods overlap at any point
// Returns false if the periods are exactly sequential, e.g.: [1, 2] and [2, 3]
func (p Period) Overlaps(other Period) bool {
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
func (p Period) OverlapsInclusive(other Period) bool {
	return p.ContainsInclusive(other.From) || p.ContainsInclusive(other.To) || other.ContainsInclusive(p.From) || other.ContainsInclusive(p.To)
}
