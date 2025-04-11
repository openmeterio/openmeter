package timeutil

import "time"

type Period interface {
	// Inclusive at both start and end
	ContainsInclusive(t time.Time) bool
	// Exclusive at both start and end
	ContainsExclusive(t time.Time) bool
	// Inclusive at start, exclusive at end
	Contains(t time.Time) bool
}
