package streaming

import (
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

type QueryParams struct {
	From           *time.Time
	To             *time.Time
	Subject        []string
	GroupBySubject bool
	GroupBy        []string
	Aggregation    models.MeterAggregation
	WindowSize     *models.WindowSize
}

func (p *QueryParams) Validate(meterWindowSize models.WindowSize) error {
	if p.From != nil && p.To != nil {
		if p.From.After(*p.To) {
			return errors.New("from must be before to")
		}
		if p.From.Equal(*p.To) {
			return errors.New("from cannot be equal with to")
		}
	}

	// Ensure `from` and `to` aligns with query param window size if any
	if p.WindowSize != nil {
		err := isRoundedToWindowSize(*p.WindowSize, p.From, p.To)
		if err != nil {
			return fmt.Errorf("cannot query with %s: %w", *p.WindowSize, err)
		}
	}

	// Ensure `from` and `to` aligns with meter aggregation window size
	err := isRoundedToWindowSize(meterWindowSize, p.From, p.To)
	if err != nil {
		return fmt.Errorf("cannot query meter aggregating on %s: %w", meterWindowSize, err)
	}

	return nil
}

func isRoundedToWindowSize(windowSize models.WindowSize, from *time.Time, to *time.Time) error {
	switch windowSize {
	case models.WindowSizeMinute:
		if from != nil && !isMinuteRounded(*from) {
			return fmt.Errorf("from must be rounded to MINUTE like XX:XX:00")
		}
		if to != nil && !isMinuteRounded(*to) {
			return fmt.Errorf("to must be rounded to MINUTE like XX:XX:00")
		}
	case models.WindowSizeHour:
		if from != nil && !isHourRounded(*from) {
			return fmt.Errorf("from must be rounded to HOUR like XX:00:00")
		}
		if to != nil && !isHourRounded(*to) {
			return fmt.Errorf("to must be rounded to HOUR like XX:00:00")
		}
	case models.WindowSizeDay:
		if from != nil && !isDayRounded(*from) {
			return fmt.Errorf("from must be rounded to DAY like 00:00:00")
		}
		if to != nil && !isDayRounded(*to) {
			return fmt.Errorf("to must be rounded to DAY like 00:00:00")
		}
	default:
		return fmt.Errorf("unknown window size %s", windowSize)
	}

	return nil
}

func isMinuteRounded(t time.Time) bool {
	return t.Second() == 0
}

func isHourRounded(t time.Time) bool {
	return isMinuteRounded(t) && t.Minute() == 0
}

func isDayRounded(t time.Time) bool {
	return isMinuteRounded(t) && isHourRounded(t) && t.Hour() == 0
}
