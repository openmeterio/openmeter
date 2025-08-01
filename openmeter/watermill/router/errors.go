package router

import "fmt"

type WarningLogSeverityError struct {
	err error
}

func NewWarningLogSeverityError(err error) error {
	return &WarningLogSeverityError{err: err}
}

func (e *WarningLogSeverityError) Error() string {
	return fmt.Sprintf("warning: %s", e.err.Error())
}

func (e *WarningLogSeverityError) Unwrap() error {
	return e.err
}
