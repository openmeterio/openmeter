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

package recurrence

import (
	"fmt"
	"time"
)

type Recurrence struct {
	Interval RecurrenceInterval `json:"period"`
	// Anchor can be an arbitrary anchor time for the recurrence.
	// It does not have to be the last or the next time.
	Anchor time.Time `json:"anchor"`
}

// NextAfter returns the next time after t that the recurrence should occur.
// If at t the recurrence should occur, it will return t.
func (r Recurrence) NextAfter(t time.Time) (time.Time, error) {
	i := r.Anchor
	if i.After(t) {
		for i.After(t) {
			v, err := r.Prev(i)
			if err != nil {
				return time.Time{}, err
			}
			i = v
		}
	}
	for i.Before(t) {
		v, err := r.Next(i)
		if err != nil {
			return time.Time{}, err
		}
		i = v
	}

	return i, nil
}

// PrevBefore returns the previous time before t that the recurrence should occur.
func (r Recurrence) PrevBefore(t time.Time) (time.Time, error) {
	i := r.Anchor
	if i.Before(t) {
		for i.Before(t) {
			v, err := r.Next(i)
			if err != nil {
				return time.Time{}, err
			}
			i = v
		}
	}
	for i.After(t) || i.Equal(t) {
		v, err := r.Prev(i)
		if err != nil {
			return time.Time{}, err
		}
		i = v
	}

	return i, nil
}

func (r Recurrence) Next(t time.Time) (time.Time, error) {
	switch r.Interval {
	case RecurrencePeriodDaily:
		return t.AddDate(0, 0, 1), nil
	case RecurrencePeriodWeek:
		return t.AddDate(0, 0, 7), nil
	case RecurrencePeriodMonth:
		return t.AddDate(0, 1, 0), nil
	case RecurrencePeriodYear:
		return t.AddDate(1, 0, 0), nil
	}
	return time.Time{}, fmt.Errorf("not implemented RecurrencePeriod %s", r.Interval)
}

func (r Recurrence) Prev(t time.Time) (time.Time, error) {
	switch r.Interval {
	case RecurrencePeriodDaily:
		return t.AddDate(0, 0, -1), nil
	case RecurrencePeriodWeek:
		return t.AddDate(0, 0, -7), nil
	case RecurrencePeriodMonth:
		return t.AddDate(0, -1, 0), nil
	case RecurrencePeriodYear:
		return t.AddDate(-1, 0, 0), nil
	}
	return time.Time{}, fmt.Errorf("not implemented RecurrencePeriod %s", r.Interval)
}

type RecurrenceInterval string

const (
	RecurrencePeriodDaily RecurrenceInterval = "DAY"
	RecurrencePeriodWeek  RecurrenceInterval = "WEEK"
	RecurrencePeriodMonth RecurrenceInterval = "MONTH"
	RecurrencePeriodYear  RecurrenceInterval = "YEAR"
)

func (RecurrenceInterval) Values() (kinds []string) {
	for _, s := range []RecurrenceInterval{
		RecurrencePeriodDaily,
		RecurrencePeriodWeek,
		RecurrencePeriodMonth,
		RecurrencePeriodYear,
	} {
		kinds = append(kinds, string(s))
	}
	return
}

func (rp RecurrenceInterval) IsValid() bool {
	switch rp {
	case RecurrencePeriodDaily,
		RecurrencePeriodWeek,
		RecurrencePeriodMonth,
		RecurrencePeriodYear:
		return true
	}
	return false
}
