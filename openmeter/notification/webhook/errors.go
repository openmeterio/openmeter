package webhook

import (
	"errors"
)

var ErrNotImplemented = errors.New("not implemented")

func IsNotImplemented(err error) bool {
	return errors.Is(err, ErrNotImplemented)
}

func IgnoreNotImplemented(err error) error {
	if IsNotImplemented(err) {
		return nil
	}

	return err
}

var _ error = (*ValidationError)(nil)

type ValidationError struct {
	err error
}

func (e ValidationError) Error() string {
	return e.err.Error()
}

func (e ValidationError) Unwrap() error {
	return e.err
}

func NewValidationError(err error) error {
	if err == nil {
		return nil
	}

	return ValidationError{err: err}
}

func IsValidationError(err error) bool {
	return isError[ValidationError](err)
}

type NotFoundError struct {
	err error
}

func (e NotFoundError) Error() string {
	return e.err.Error()
}

func (e NotFoundError) Unwrap() error {
	return e.err
}

func NewNotFoundError(err error) error {
	if err == nil {
		return nil
	}

	return NotFoundError{err: err}
}

func IsNotFoundError(err error) bool {
	return isError[NotFoundError](err)
}

func isError[T error](err error) bool {
	var t T

	return errors.As(err, &t)
}
