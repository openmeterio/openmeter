package meter

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
)

// NewMeterNotFoundError returns a new MeterNotFoundError.
func NewMeterNotFoundError(meterSlug string) error {
	return &MeterNotFoundError{
		err: models.NewGenericNotFoundError(
			fmt.Errorf("meter not found: %s", meterSlug),
		),
	}
}

var _ models.GenericError = &MeterNotFoundError{}

// MeterNotFoundError is returned when a meter is not found.
type MeterNotFoundError struct {
	err error
}

// Error returns the error message.
func (e *MeterNotFoundError) Error() string {
	return e.err.Error()
}

// Unwrap returns the wrapped error.
func (e *MeterNotFoundError) Unwrap() error {
	return e.err
}

// IsMeterNotFoundError returns true if the error is a MeterNotFoundError.
func IsMeterNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	var e *MeterNotFoundError

	return errors.As(err, &e)
}
