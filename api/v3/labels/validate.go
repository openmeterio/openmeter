package labels

import (
	"errors"
	"fmt"
	"regexp"

	api "github.com/openmeterio/openmeter/api/v3"
)

var (
	ErrInvalidLabelKey   = errors.New("invalid label key")
	ErrInvalidLabelValue = errors.New("invalid label value")
)

// https://regex101.com/?regex=%5E%28%3F%3A%5Ba-zA-Z0-9%5D%28%3F%3A%5Ba-zA-Z0-9._-%5D%5Ba-zA-Z0-9%5D%29%3F%29%7B1%2C63%7D%24&testString=openmeter_good%0Aopenmeter.good%0Aopenmeter-good%0Aopenmeter_bad_%0Aopenmeter-bad-%0Aopenmeter.bad.%0A_openmeter-bad%0A-openmeter-bad%0A.openmeter.bad%0A.openmeter.way_toooooooooooooooooooooooooooooooooooooooooooooooooooooo_long&flags=gm&flavor=pcre2&delimiter=%2F
var keyValueFormat = regexp.MustCompile(`^(?:[a-zA-Z0-9](?:[a-zA-Z0-9._-]?[a-zA-Z0-9])?){1,63}$`)

func ValidateLabel(k, v string) error {
	var errs []error

	if !keyValueFormat.MatchString(k) {
		errs = append(errs, ErrInvalidLabelKey)
	}

	if !keyValueFormat.MatchString(v) {
		errs = append(errs, ErrInvalidLabelValue)
	}

	return errors.Join(errs...)
}

func ValidateLabels(l api.Labels) error {
	var errs []error

	for k, v := range l {
		err := ValidateLabel(k, v)
		if err != nil {
			if errors.Is(err, ErrInvalidLabelKey) {
				errs = append(errs, fmt.Errorf("%w: %q", err, k))
			}

			if errors.Is(err, ErrInvalidLabelValue) {
				errs = append(errs, fmt.Errorf("%w: %q", err, v))
			}
		}
	}

	return errors.Join(errs...)
}
