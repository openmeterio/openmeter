package datetime

import (
	"fmt"
)

// NewInvalidTimezoneError creates an error for when an invalid timezone is specified.
func NewInvalidTimezoneError(timezone string, err error) error {
	return fmt.Errorf("invalid timezone '%s': %w", timezone, err)
}

// NewDateTimeParseError creates a general parse error for datetime values.
func NewDateTimeParseError(value string) error {
	return fmt.Errorf("failed to parse datetime '%s'", value)
}

// NewDurationParseError creates a general parse error for duration values.
func NewDurationParseError(value string, err error) error {
	return fmt.Errorf("failed to parse duration '%s': %w", value, err)
}
