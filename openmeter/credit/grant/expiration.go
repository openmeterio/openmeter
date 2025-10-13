package grant

import "time"

// ExpirationPeriod of a credit grant.
type ExpirationPeriod struct {
	// Count The expiration period count like 12 months.
	Count uint32 `json:"count,omitempty"`

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
	return kinds
}
