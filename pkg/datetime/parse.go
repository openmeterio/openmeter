package datetime

import (
	"errors"
	"strings"
	"time"
)

// Parse parses a RFC3339, ISO8601, or RFC9557 formatted string into a DateTime.
func Parse(value string) (DateTime, error) {
	if value == "" {
		return DateTime{}, NewDateTimeParseError(value)
	}

	// Check if the value has a timezone suffix
	i := strings.LastIndexByte(value, '[')
	if i == -1 || !strings.HasSuffix(value, "]") {
		// No timezone suffix, try standard layouts
		layouts := []string{
			time.RFC3339,
			time.RFC3339Nano,
			ISO8601Layout,
			ISO8601MilliLayout,
			ISO8601MicroLayout,
			ISO8601NanoLayout,
			ISO8601ZuluLayout,
			ISO8601ZuluMilliLayout,
			ISO8601ZuluMicroLayout,
			ISO8601ZuluNanoLayout,
		}

		for _, layout := range layouts {
			if t, err := time.Parse(layout, value); err == nil {
				return DateTime{t}, nil
			}
		}

		return DateTime{}, NewDateTimeParseError(value)
	}

	// Extract timestamp and timezone
	timestamp := value[:i]
	timezone := value[i+1 : len(value)-1]

	// Validate timezone bracket format
	if timezone == "" {
		return DateTime{}, NewInvalidTimezoneError(timezone, errors.New("timezone identifier is empty"))
	}

	// Load the timezone location
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return DateTime{}, NewInvalidTimezoneError(timezone, err)
	}

	// Try parsing with RFC9557-compatible layouts
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		ISO8601ZuluLayout,
		ISO8601ZuluMilliLayout,
		ISO8601ZuluMicroLayout,
		ISO8601ZuluNanoLayout,
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, timestamp); err == nil {
			// Convert to the specified timezone
			return DateTime{t.In(loc)}, nil
		}
	}

	return DateTime{}, NewDateTimeParseError(value)
}
