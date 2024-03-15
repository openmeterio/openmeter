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

package grant

import "time"

// ExpirationPeriod of a credit grant.
type ExpirationPeriod struct {
	// Count The expiration period count like 12 months.
	Count uint8 `json:"count,omitempty"`

	// Duration The expiration period duration like month.
	Duration ExpirationPeriodDuration `json:"duration,omitempty"`
}

func (c ExpirationPeriod) GetExpiration(t time.Time) time.Time {
	switch c.Duration {
	case ExpirationPeriodDurationHour:
		return t.Add(time.Hour * time.Duration(c.Count))
	case ExpirationPeriodDurationDay:
		return t.AddDate(0, 0, int(c.Count))
	case ExpirationPeriodDurationWeek:
		return t.AddDate(0, 0, int(c.Count*7))
	case ExpirationPeriodDurationMonth:
		return t.AddDate(0, int(c.Count), 0)
	case ExpirationPeriodDurationYear:
		return t.AddDate(int(c.Count), 0, 0)
	default:
		return time.Time{}
	}
}

type ExpirationPeriodDuration string

// Defines values for ExpirationPeriodDuration.
const (
	ExpirationPeriodDurationHour  ExpirationPeriodDuration = "HOUR"
	ExpirationPeriodDurationDay   ExpirationPeriodDuration = "DAY"
	ExpirationPeriodDurationWeek  ExpirationPeriodDuration = "WEEK"
	ExpirationPeriodDurationMonth ExpirationPeriodDuration = "MONTH"
	ExpirationPeriodDurationYear  ExpirationPeriodDuration = "YEAR"
)

func (ExpirationPeriodDuration) Values() (kinds []string) {
	for _, s := range []ExpirationPeriodDuration{
		ExpirationPeriodDurationHour,
		ExpirationPeriodDurationDay,
		ExpirationPeriodDurationWeek,
		ExpirationPeriodDurationMonth,
		ExpirationPeriodDurationYear,
	} {
		kinds = append(kinds, string(s))
	}
	return
}
