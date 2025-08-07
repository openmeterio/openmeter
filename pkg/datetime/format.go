package datetime

import "strings"

// Format extends the time.Time.Format method to support the RFC 9557 format
func (t DateTime) Format(layout string) string {
	timezone := t.Location().String()

	// If the timezone is not set use the fallback layout
	if timezone == "" {
		return t.Time.Format(fallbackRFC9557FormatLayout(layout))
	}

	// Replace the timezone name layout part with the timezone name
	return strings.ReplaceAll(t.Time.Format(layout), layoutTZName, timezone)
}

// fallbackFormatLayout can be used to convert a RFC9557 layout to a ISO8601 layout when no timezone is provided
func fallbackRFC9557FormatLayout(layout string) string {
	switch layout {
	case RFC9557Layout:
		return ISO8601Layout
	case RFC9557MilliLayout:
		return ISO8601MilliLayout
	case RFC9557MicroLayout:
		return ISO8601MicroLayout
	case RFC9557NanoLayout:
		return ISO8601NanoLayout
	default:
		return layout
	}
}
