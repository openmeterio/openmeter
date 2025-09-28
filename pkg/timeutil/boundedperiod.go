package timeutil

import (
	"errors"
	"time"
)

type StartBoundedPeriod struct {
	From time.Time  `json:"from"`
	To   *time.Time `json:"to,omitempty"`
}

var _ Period = StartBoundedPeriod{}

func (p StartBoundedPeriod) Validate() error {
	if p.From.IsZero() {
		return errors.New("from is required")
	}

	if p.To != nil && p.From.After(*p.To) {
		return errors.New("from must be before to")
	}

	return nil
}

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
