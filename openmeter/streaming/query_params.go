package streaming

import (
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

type QueryParams struct {
	ClientID       *string
	From           *time.Time
	To             *time.Time
	FilterSubject  []string
	FilterGroupBy  map[string][]string
	GroupBy        []string
	WindowSize     *models.WindowSize
	WindowTimeZone *time.Location
}

// Validate validates query params focusing on `from` and `to` being aligned with query and meter window sizes
func (p *QueryParams) Validate(meter models.Meter) error {
	if p.ClientID != nil && len(*p.ClientID) == 0 {
		return errors.New("client id cannot be empty")
	}

	if p.From != nil && p.To != nil {
		if !p.To.After(*p.From) {
			return errors.New("to must be after from")
		}
	}

	if err := meter.SupportsWindowSize(p.WindowSize); err != nil {
		return err
	}

	// Ensure `from` and `to` aligns with meter aggregation window size
	err := isRoundedToWindowSize(meter.WindowSize, p.From, p.To)
	if err != nil {
		return fmt.Errorf("cannot query meter aggregating on %s window size: %w", meter.WindowSize, err)
	}

	return nil
}

// Checks if `from` and `to` are rounded to window size
func isRoundedToWindowSize(windowSize models.WindowSize, from *time.Time, to *time.Time) error {
	switch windowSize {
	case models.WindowSizeMinute:
		if from != nil && !isMinuteRounded(from.UTC()) {
			return fmt.Errorf("from must be rounded to MINUTE like YYYY-MM-DDTHH:mm:00")
		}
		if to != nil && !isMinuteRounded(to.UTC()) {
			return fmt.Errorf("to must be rounded to MINUTE like YYYY-MM-DDTHH:mm:00")
		}
	case models.WindowSizeHour:
		if from != nil && !isHourRounded(from.UTC()) {
			return fmt.Errorf("from must be rounded to HOUR like YYYY-MM-DDTHH:00:00")
		}
		if to != nil && !isHourRounded(to.UTC()) {
			return fmt.Errorf("to must be rounded to HOUR like YYYY-MM-DDTHH:00:00")
		}
	case models.WindowSizeDay:
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

type ListEventsParams struct {
	ClientID       *string
	From           time.Time
	To             *time.Time
	IngestedAtFrom *time.Time
	IngestedAtTo   *time.Time
	ID             *string
	Subject        *string
	HasError       *bool
	Limit          int
}

func (p ListEventsParams) Validate(minimumFrom time.Time) error {
	var errs []error

	if p.ClientID != nil && *p.ClientID == "" {
		errs = append(errs, errors.New("clientID is empty"))
	}

	if p.From.Before(minimumFrom) {
		errs = append(errs, fmt.Errorf("from date is too old: %s", p.From))
	}

	if p.To != nil && p.To.Before(p.From) {
		errs = append(errs, fmt.Errorf("to date is before from date: %s < %s", p.To, p.From))
	}

	if p.IngestedAtFrom != nil && p.IngestedAtFrom.Before(minimumFrom) {
		errs = append(errs, fmt.Errorf("ingestedAtFrom date is too old: %s", p.IngestedAtFrom))
	}

	if p.IngestedAtFrom != nil && p.IngestedAtTo != nil && p.IngestedAtTo.Before(*p.IngestedAtFrom) {
		errs = append(errs, fmt.Errorf("ingestedAtTo date is before ingestedAtFrom date: %s < %s", p.IngestedAtTo, p.IngestedAtFrom))
	}

	if p.Limit <= 0 {
		errs = append(errs, errors.New("limit must be greater than 0"))
	}

	if p.ID != nil && *p.ID == "" {
		errs = append(errs, errors.New("id is empty"))
	}

	if p.Subject != nil && *p.Subject == "" {
		errs = append(errs, errors.New("subject is empty"))
	}

	return errors.Join(errs...)
}

type CountEventsParams struct {
	From time.Time
}

type ListMeterSubjectsParams struct {
	From *time.Time
	To   *time.Time
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
