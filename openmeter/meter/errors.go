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

// NewBadRequestError returns a new BadRequestError.
func NewBadRequestError(err error) error {
	return &BadRequestError{err: err}
}

// BadRequestError is returned when a meter is not found.
type BadRequestError struct {
	err error
}

// Error returns the error message.
func (e *BadRequestError) Error() string {
	return fmt.Sprintf("bad request: %s", e.err)
}

// IsBadRequestError returns true if the error is a BadRequestError.
func IsBadRequestError(err error) bool {
	if err == nil {
		return false
	}

	var e *BadRequestError

	return errors.As(err, &e)
}
