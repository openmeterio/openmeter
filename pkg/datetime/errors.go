package datetime

import (
	"github.com/samber/oops"
)

const (
	DateTimeErrorDomain = "datetime"

	DateTimeInvalidTimezoneErrorCode = "datetime_invalid_timezone"
	DateTimeParseErrorCode           = "datetime_parse_failed"
	DurationParseErrorCode           = "duration_parse_failed"
)

// NewInvalidTimezoneError creates an error for when an invalid timezone is specified.
func NewInvalidTimezoneError(timezone string, err error) error {
	return oops.
		In(DateTimeErrorDomain).
		Code(DateTimeInvalidTimezoneErrorCode).
		With("timezone", timezone).
		Hint("Use a valid IANA timezone identifier").
		Wrapf(err, "invalid timezone '%s'", timezone)
}

// NewDateTimeParseError creates a general parse error for datetime values.
func NewDateTimeParseError(value string) error {
	return oops.
		In(DateTimeErrorDomain).
		Code(DateTimeParseErrorCode).
		With("input", value).
		Hint("Input must be in RFC3339, ISO8601, or RFC9557 format").
		Errorf("failed to parse datetime: %s", value)
}

// NewDurationParseError creates a general parse error for duration values.
func NewDurationParseError(value string, err error) error {
	return oops.
		In(DateTimeErrorDomain).
		Code(DurationParseErrorCode).
		With("input", value).
		Hint("Input must be in ISO8601 duration format").
		Errorf("failed to parse duration: %s", value)
}
