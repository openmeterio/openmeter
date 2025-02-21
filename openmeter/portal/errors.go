package portal

import (
	"errors"
)

// NotImplementedError represents an error when not implemented
var _ error = (*NotImplementedError)(nil)

// NewNotImplementedError returns a new not implemented error
func NewNotImplementedError(err error) *NotImplementedError {
	return &NotImplementedError{
		Err: err,
	}
}

// NotImplementedError represents an error when not implemented
type NotImplementedError genericError

func (e NotImplementedError) Error() string {
	return e.Err.Error()
}

// genericError represents a generic error
type genericError struct {
	Err error
}

// IsNotImplemented returns a boolean indicating whether the error is a not implemented error.
func IsNotImplemented(err error) bool {
	if err == nil {
		return false
	}

	var e *NotImplementedError

	return errors.As(err, &e)
}
