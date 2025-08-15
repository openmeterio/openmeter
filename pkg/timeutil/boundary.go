package timeutil

import (
	"fmt"
	"slices"
)

type Boundary string

const (
	// Exclusive means the specified boundary time is excluded from the interval.
	// For example, for the range (2025-01-01T00:00:00Z, 2025-01-02T00:00:00Z),
	// the instant 2025-01-01T00:00:00Z is not considered part of the interval.
	Exclusive Boundary = "exclusive"

	// Inclusive means the specified boundary time is included in the interval.
	// For example, for the range [2025-01-01T00:00:00Z, 2025-01-02T00:00:00Z],
	// the instant 2025-01-01T00:00:00Z is considered part of the interval.
	Inclusive Boundary = "inclusive"
)

func (b Boundary) Validate() error {
	if !slices.Contains([]Boundary{Exclusive, Inclusive}, b) {
		return fmt.Errorf("invalid boundary type: %s", b)
	}

	return nil
}
