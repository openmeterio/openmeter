// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	FilterSubject  []string
	FilterGroupBy  map[string][]string
	GroupBy        []string
	Aggregation    models.MeterAggregation
	WindowSize     *models.WindowSize
	WindowTimeZone *time.Location
}

// Validate validates query params focusing on `from` and `to` being aligned with query and meter window sizes
func (p *QueryParams) Validate(meter models.Meter) error {
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
