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

// Inclusive at start, exclusive at end
func (p Period) Contains(t time.Time) bool {
	return (t.After(p.From) || t.Equal(p.From)) && t.Before(p.To)
}
