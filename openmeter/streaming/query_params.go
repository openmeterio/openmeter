package streaming

import (
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/models"
)

type QueryParams struct {
	From           *time.Time
	To             *time.Time
	FilterSubject  []string
	FilterGroupBy  map[string][]string
	GroupBy        []string
	WindowSize     *meter.WindowSize
	WindowTimeZone *time.Location
}

// Validate validates query params focusing on `from` and `to` being aligned with query and meter window sizes
func (p *QueryParams) Validate(meter meter.Meter) error {
	var errs []error

	if p.From != nil && p.To != nil {
		if p.From.Equal(*p.To) {
			errs = append(errs, errors.New("from and to cannot be equal"))
		}

		if p.From.After(*p.To) {
			errs = append(errs, errors.New("from must be before to"))
		}
	}

	if err := meter.SupportsWindowSize(p.WindowSize); err != nil {
		errs = append(errs, err)
	}

	// Ensure `from` and `to` aligns with meter aggregation window size
	err := isRoundedToWindowSize(meter.WindowSize, p.From, p.To)
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return models.NewNillableGenericValidationError(errors.Join(errs...))
	}

	return nil
}

// Checks if `from` and `to` are rounded to window size
func isRoundedToWindowSize(windowSize meter.WindowSize, from *time.Time, to *time.Time) error {
	switch windowSize {
	case meter.WindowSizeMinute:
		if from != nil && !isMinuteRounded(from.UTC()) {
			return fmt.Errorf("from must be rounded to MINUTE like YYYY-MM-DDTHH:mm:00")
		}
		if to != nil && !isMinuteRounded(to.UTC()) {
			return fmt.Errorf("to must be rounded to MINUTE like YYYY-MM-DDTHH:mm:00")
		}
	case meter.WindowSizeHour:
		if from != nil && !isHourRounded(from.UTC()) {
			return fmt.Errorf("from must be rounded to HOUR like YYYY-MM-DDTHH:00:00")
		}
		if to != nil && !isHourRounded(to.UTC()) {
			return fmt.Errorf("to must be rounded to HOUR like YYYY-MM-DDTHH:00:00")
		}
	case meter.WindowSizeDay:
		if from != nil && !isDayRounded(from.UTC()) {
			return fmt.Errorf("from must be rounded to DAY like YYYY-MM-DDT00:00:00")
		}
		if to != nil && !isDayRounded(to.UTC()) {
			return fmt.Errorf("to must be rounded to DAY like YYYY-MM-DDT00:00:00")
		}
	default:
		return fmt.Errorf("unknown window size %s", windowSize)
	}

	return nil
}

// Is rounded to minute like YYYY-MM-DDTHH:mm:00
func isMinuteRounded(t time.Time) bool {
	return t.Second() == 0
}

// Is rounded to hour like YYYY-MM-DDTHH:00:00
func isHourRounded(t time.Time) bool {
	return t.Second() == 0 && t.Minute() == 0
}

// Is rounded to day like YYYY-MM-DDT00:00:00
func isDayRounded(t time.Time) bool {
	return t.Second() == 0 && t.Minute() == 0 && t.Hour() == 0
}
