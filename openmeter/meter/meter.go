package meter

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

var groupByKeyRegExp = regexp.MustCompile(`^[a-zA-Z_][0-9a-zA-Z_]*$`)

type MeterAggregation string

// Note: keep values up to date in the meter package
const (
	MeterAggregationSum         MeterAggregation = "SUM"
	MeterAggregationCount       MeterAggregation = "COUNT"
	MeterAggregationAvg         MeterAggregation = "AVG"
	MeterAggregationMin         MeterAggregation = "MIN"
	MeterAggregationMax         MeterAggregation = "MAX"
	MeterAggregationUniqueCount MeterAggregation = "UNIQUE_COUNT"
)

// Values provides list valid values for Enum
func (MeterAggregation) Values() (kinds []string) {
	for _, s := range []MeterAggregation{
		MeterAggregationSum,
		MeterAggregationCount,
		MeterAggregationAvg,
		MeterAggregationMin,
		MeterAggregationMax,
		MeterAggregationUniqueCount,
	} {
		kinds = append(kinds, string(s))
	}
	return
}

func (MeterAggregation) IsValid(input string) bool {
	m := MeterAggregation("")

	for _, v := range m.Values() {
		if v == input {
			return true
		}
	}

	return false
}

type WindowSize string

// Note: keep values up to date in the meter package
const (
	WindowSizeMinute WindowSize = "MINUTE"
	WindowSizeHour   WindowSize = "HOUR"
	WindowSizeDay    WindowSize = "DAY"
)

// Values provides list valid values for Enum
func (WindowSize) Values() (kinds []string) {
	for _, s := range []WindowSize{
		WindowSizeMinute,
		WindowSizeHour,
		WindowSizeDay,
	} {
		kinds = append(kinds, string(s))
	}
	return
}

func (w WindowSize) AddTo(t time.Time) (time.Time, error) {
	switch w {
	case WindowSizeMinute:
		return t.Add(time.Minute), nil
	case WindowSizeHour:
		return t.Add(time.Hour), nil
	case WindowSizeDay:
		return t.AddDate(0, 0, 1), nil
	default:
		return time.Time{}, fmt.Errorf("invalid window size: %s", w)
	}
}

func (w WindowSize) Truncate(t time.Time) (time.Time, error) {
	switch w {
	case WindowSizeMinute:
		return t.Truncate(time.Minute), nil
	case WindowSizeHour:
		return t.Truncate(time.Hour), nil
	case WindowSizeDay:
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()), nil
	default:
		return time.Time{}, fmt.Errorf("invalid window size: %s", w)
	}
}

// Duration returns the duration of the window size
// BEWARE: a day is NOT 24 hours
func (w WindowSize) Duration() time.Duration {
	var windowDuration time.Duration
	switch w {
	case WindowSizeMinute:
		windowDuration = time.Minute
	case WindowSizeHour:
		windowDuration = time.Hour
	case WindowSizeDay:
		windowDuration = 24 * time.Hour
	}

	return windowDuration
}

func WindowSizeFromDuration(duration time.Duration) (WindowSize, error) {
	switch duration.Minutes() {
	case time.Minute.Minutes():
		return WindowSizeMinute, nil
	case time.Hour.Minutes():
		return WindowSizeHour, nil
	case 24 * time.Hour.Minutes():
		return WindowSizeDay, nil
	default:
		return "", fmt.Errorf("invalid window size duration: %s", duration)
	}
}

type Meter struct {
	Namespace     string
	ID            string
	Slug          string
	Description   string
	Aggregation   MeterAggregation
	EventType     string
	ValueProperty string
	GroupBy       map[string]string
	WindowSize    WindowSize
}

type MeterOptions struct {
	ID          string
	Description string
	GroupBy     map[string]string
	WindowSize  *WindowSize
}

func NewMeter(
	slug string,
	aggregatation MeterAggregation,
	eventType string,
	valueProperty string,
	options *MeterOptions,
) (*Meter, error) {
	meter := &Meter{
		Slug:          slug,
		Aggregation:   aggregatation,
		EventType:     eventType,
		ValueProperty: valueProperty,
	}

	// Apply optional parameters if provided
	if options != nil {
		meter.ID = options.ID
		meter.Description = options.Description
		meter.GroupBy = options.GroupBy
		if options.WindowSize != nil {
			meter.WindowSize = *options.WindowSize
		}
	}

	err := meter.Validate()
	if err != nil {
		return nil, err
	}

	return meter, nil
}

func (m *Meter) Validate() error {
	if m.ID != "" {
		_, err := ulid.ParseStrict(m.ID)
		if err != nil {
			return errors.New("meter id is not a valid ULID")
		}
	}
	if m.Slug == "" {
		return errors.New("meter slug is required")
	}
	// FIXME: prefix materialized views to allow `events` as a meter slug
	// `events` is a restricted name as it conflicts with the events table
	if m.Slug == "events" {
		return errors.New("meter slug cannot be `events`")
	}
	if len(m.Slug) > 63 {
		return errors.New("meter slug must be less than 64 characters")
	}
	if m.EventType == "" {
		return errors.New("meter event type is required")
	}
	if m.Aggregation == "" {
		return errors.New("meter aggregation is required")
	}

	// ValueProperty is required for all aggregations except count
	if m.Aggregation == MeterAggregationCount {
		if m.ValueProperty != "" {
			return errors.New("meter value property is not allowed when the aggregation is count")
		}
	} else {
		if m.ValueProperty == "" {
			return errors.New("meter value property is required when the aggregation is not count")
		}

		if !strings.HasPrefix(m.ValueProperty, "$") {
			return errors.New("meter value property must start with $")
		}
	}

	for key, value := range m.GroupBy {
		if !strings.HasPrefix(value, "$") {
			return fmt.Errorf("meter group by value must start with $ for key %s", key)
		}
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("meter group by key cannot be empty")
		}
		if !groupByKeyRegExp.MatchString(key) {
			return fmt.Errorf("meter group by key %s is invalid, only alphanumeric and underscore characters are allowed", key)
		}
		if value == m.ValueProperty {
			return fmt.Errorf("meter group by value %s cannot be the same as value property", key)
		}
		// keys must be unique
		seen := make(map[string]struct{}, len(m.GroupBy))
		if _, ok := seen[key]; ok {
			return fmt.Errorf("meter group by key %s is not unique", key)
		}
		seen[key] = struct{}{}
	}

	return nil
}

func (m *Meter) SupportsWindowSize(w *WindowSize) error {
	// Ensure `from` and `to` aligns with query param window size if any
	if w != nil {
		// Ensure query param window size is not smaller than meter window size
		switch m.WindowSize {
		case WindowSizeHour:
			if w != nil && *w == WindowSizeMinute {
				return fmt.Errorf("cannot query meter with window size %s on window size %s", m.WindowSize, *w)
			}
		case WindowSizeDay:
			if w != nil && (*w == WindowSizeMinute || *w == WindowSizeHour) {
				return fmt.Errorf("cannot query meter with window size %s on window size %s", m.WindowSize, *w)
			}
		}
	}
	return nil
}

func (m *Meter) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// MeterQueryRow returns a single row from the meter dataset.
type MeterQueryRow struct {
	Value       float64            `json:"value"`
	WindowStart time.Time          `json:"windowStart"`
	WindowEnd   time.Time          `json:"windowEnd"`
	Subject     *string            `json:"subject"`
	GroupBy     map[string]*string `json:"groupBy"`
}

// Render implements the chi renderer interface.
func (m *MeterQueryRow) Render(_ http.ResponseWriter, _ *http.Request) error {
	return nil
}
