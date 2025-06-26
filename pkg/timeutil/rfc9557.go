package timeutil

import (
	"fmt"
	"regexp"
	"time"
)

type RFC9557Time struct {
	// Note: This is not public so that we cannot accidentally add a time object without a controlled location
	t time.Time
}

var rfc9557LayoutRegex = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})(\.\d+)?\[(.+)\]$`)

const (
	rfc3339WithoutTZ     = "2006-01-02T15:04:05"
	rfc3339NanoWithoutTZ = "2006-01-02T15:04:05.999999999"
)

// ParseRFC9557 parses a RFC9557 timestamp string into a RFC9557Time object.
// Limitations/behavior:
// - If the timestamp is a valid RFC3339 timestamp it is normalized to UTC and location is UTC
// - If the timestamp is a valid RFC9557 timestamp with location it is parsed as is, the location is validated
// - We are not supporting RFC9557 timestamps `[!x=y]` formatted attributes
func ParseRFC9557(s string) (RFC9557Time, error) {
	// Let's try to parse as RFC3339 first
	res, err := time.Parse(time.RFC3339, s)
	if err == nil {
		if res.Location() == time.UTC {
			return RFC9557Time{res}, nil
		}

		// Location is not in UTC, but RFC3339 only supports tz offsets which is not acceptable for us
		// so let's normalize the time to UTC and assume the intent of UTC location
		return RFC9557Time{
			t: res.In(time.UTC),
		}, nil
	}

	matches := rfc9557LayoutRegex.FindStringSubmatch(s)
	if len(matches) < 3 {
		return RFC9557Time{}, fmt.Errorf("invalid RFC 9557 timestamp: %s", s)
	}

	if len(matches) == 4 {
		// Nano timestamp
		loc, err := time.LoadLocation(matches[3])
		if err != nil {
			return RFC9557Time{}, err
		}

		timeWithoutTZ := matches[1] + matches[2]

		res, err := time.ParseInLocation(rfc3339NanoWithoutTZ, timeWithoutTZ, loc)
		if err != nil {
			return RFC9557Time{}, err
		}

		return RFC9557Time{res}, nil
	}

	// Normal timestamp without subsecond
	loc, err := time.LoadLocation(matches[2])
	if err != nil {
		return RFC9557Time{}, err
	}

	res, err = time.ParseInLocation(rfc3339WithoutTZ, matches[1], loc)
	if err != nil {
		return RFC9557Time{}, err
	}

	return RFC9557Time{res}, nil
}

func (t RFC9557Time) String() string {
	if t.t.Location() == time.UTC {
		return t.t.Format(time.RFC3339Nano)
	}

	return fmt.Sprintf("%s[%s]", t.t.Format(rfc3339NanoWithoutTZ), t.t.Location().String())
}

func (t RFC9557Time) Time() time.Time {
	return t.t
}

func (t RFC9557Time) Location() *time.Location {
	return t.t.Location()
}
