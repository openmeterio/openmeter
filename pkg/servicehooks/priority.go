package servicehooks

import "fmt"

// Priority controls hook invocation order. Lower values run first.
type Priority int

const (
	PriorityHighest Priority = 0
	PriorityHigh    Priority = 25
	PriorityDefault Priority = 50
	PriorityLow     Priority = 75
	PriorityLowest  Priority = 100
)

// Validate reports whether the priority is within the supported range.
func (p Priority) Validate() error {
	if p < PriorityHighest || p > PriorityLowest {
		return fmt.Errorf("%w: got %d, expected %d-%d", ErrPriorityOutOfRange, p, PriorityHighest, PriorityLowest)
	}

	return nil
}
