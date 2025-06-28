package datetime

import (
	"encoding/json"
	"time"
)

// DateTime extends the time.Time type to support the RFC 9557 format.
type DateTime struct {
	time.Time
}

// NewDateTime creates a new DateTime from a time.Time.
func NewDateTime(t time.Time) DateTime {
	return DateTime{t}
}

// AsTime returns the underlying time.Time.
func (t DateTime) AsTime() time.Time {
	return t.Time
}

// MarshalJSON marshals the DateTime to a JSON string in RFC3339 format.
func (t DateTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Format(time.RFC3339))
}

// UnmarshalJSON unmarshals the DateTime from a JSON string.
// It supports RFC3339, ISO8601, and RFC9557 formats.
func (t *DateTime) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	date, err := Parse(s)
	if err != nil {
		return err
	}

	*t = date

	return nil
}

// Add adds a duration to a DateTime.
func (dt DateTime) Add(d Duration) DateTime {
	years := d.Years()
	months := d.Months()
	weeks := d.Weeks()
	days := d.Days()
	hours := d.Hours()
	minutes := d.Minutes()
	seconds := d.Seconds()

	dt = dt.AddYearsNoOverflow(years).
		AddMonthsNoOverflow(months).
		AddWeeks(weeks).
		AddDays(days).
		AddHours(hours).
		AddMinutes(minutes).
		AddSeconds(seconds)

	return dt
}

// AddDateNoOverflow adds some years, months, and days without overflowing month.
func (d DateTime) AddDateNoOverflow(years int, months int, days int) DateTime {
	d = d.AddYearsNoOverflow(years).
		AddMonthsNoOverflow(months).
		AddDays(days)

	return d
}

// AddYearsNoOverflow adds some years without overflowing month.
func (d DateTime) AddYearsNoOverflow(years int) DateTime {
	nanosecond := d.Nanosecond()
	year, month, day := d.Date()
	hour, minute, second := d.Clock()
	// get the last day of this month after some years
	lastYear, lastMonth, lastDay := time.Date(year+years, month+1, 0, hour, minute, second, nanosecond, d.Location()).Date()
	if day > lastDay {
		day = lastDay
	}
	d.Time = time.Date(lastYear, lastMonth, day, hour, minute, second, nanosecond, d.Location())
	return d
}

// AddMonthsNoOverflow adds some months without overflowing month.
func (d DateTime) AddMonthsNoOverflow(months int) DateTime {
	nanosecond := d.Nanosecond()
	year, month, day := d.Date()
	hour, minute, second := d.Clock()
	// get the last day of this month after some months
	lastYear, lastMonth, lastDay := time.Date(year, month+time.Month(months+1), 0, hour, minute, second, nanosecond, d.Location()).Date()
	if day > lastDay {
		day = lastDay
	}
	d.Time = time.Date(lastYear, lastMonth, day, hour, minute, second, nanosecond, d.Location())
	return d
}

// AddWeeks adds some weeks to the DateTime.
func (d DateTime) AddWeeks(weeks int) DateTime {
	d.Time = d.Time.AddDate(0, 0, weeks*7)
	return d
}

// AddDays adds some days to the DateTime.
func (d DateTime) AddDays(days int) DateTime {
	d.Time = d.Time.AddDate(0, 0, days)
	return d
}

// AddHours adds some hours to the DateTime.
func (d DateTime) AddHours(hours int) DateTime {
	d.Time = d.Time.Add(time.Duration(hours) * time.Hour)
	return d
}

// AddMinutes adds some minutes to the DateTime.
func (d DateTime) AddMinutes(minutes int) DateTime {
	d.Time = d.Time.Add(time.Duration(minutes) * time.Minute)
	return d
}

// AddSeconds adds some seconds to the DateTime.
func (d DateTime) AddSeconds(seconds int) DateTime {
	d.Time = d.Time.Add(time.Duration(seconds) * time.Second)
	return d
}
