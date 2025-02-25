package meter

import (
	"errors"
	"fmt"
)

// NewMeterNotFoundError returns a new MeterNotFoundError.
func NewMeterNotFoundError(meterSlug string) error {
	return &MeterNotFoundError{MeterSlug: meterSlug}
}

// MeterNotFoundError is returned when a meter is not found.
type MeterNotFoundError struct {
	MeterSlug string
}

// Error returns the error message.
func (e *MeterNotFoundError) Error() string {
	return fmt.Sprintf("meter not found: %s", e.MeterSlug)
}

// IsMeterNotFoundError returns true if the error is a MeterNotFoundError.
func IsMeterNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	var e *MeterNotFoundError

	return errors.As(err, &e)
}
