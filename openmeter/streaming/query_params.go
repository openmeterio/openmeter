package streaming

import (
	"errors"
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
func (p *QueryParams) Validate() error {
	var errs []error

	if p.From != nil && p.To != nil {
		if p.From.Equal(*p.To) {
			errs = append(errs, errors.New("from and to cannot be equal"))
		}

		if p.From.After(*p.To) {
			errs = append(errs, errors.New("from must be before to"))
		}
	}

	if len(errs) > 0 {
		return models.NewNillableGenericValidationError(errors.Join(errs...))
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
