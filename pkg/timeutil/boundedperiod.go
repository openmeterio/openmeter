package timeutil

import "time"

type StartBoundedPeriod struct {
	From time.Time  `json:"from"`
	To   *time.Time `json:"to,omitempty"`
}

var _ Period = StartBoundedPeriod{}

// Inclusive at both start and end
func (p StartBoundedPeriod) ContainsInclusive(t time.Time) bool {
	if t.Before(p.From) {
		return false
	}

	if p.To != nil && t.After(*p.To) {
		return false
	}

	return true
}

// Exclusive at both start and end
func (p StartBoundedPeriod) ContainsExclusive(t time.Time) bool {
	if t.Before(p.From) || t.Equal(p.From) {
		return false
	}

	if p.To != nil && (t.After(*p.To) || t.Equal(*p.To)) {
		return false
	}

	return true
}

// Inclusive at start, exclusive at end
func (p StartBoundedPeriod) Contains(t time.Time) bool {
	if t.Before(p.From) {
		return false
	}

	if p.To != nil && (t.After(*p.To) || t.Equal(*p.To)) {
		return false
	}

	return true
}

func (p StartBoundedPeriod) Open() OpenPeriod {
	return OpenPeriod{
		From: &p.From,
		To:   p.To,
	}
}
