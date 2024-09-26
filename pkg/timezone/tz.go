package timezone

import "time"

// TimeZone represents a timezone. Going forward we will need to provide a list of timezones to the user.
// Otherwise we will depend on whatever's available in the underlying container.
type Timezone string

func (t Timezone) LoadLocation() (*time.Location, error) {
	if t == "" {
		return time.UTC, nil
	}
	return time.LoadLocation(string(t))
}
