package labels

import (
	"errors"
	"net/http"
	"regexp"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/models"
)

var (
	ErrInvalidLabelKey   = errors.New("invalid label key")
	ErrInvalidLabelValue = errors.New("invalid label value")
)

// https://regex101.com/?regex=%5E%28%3F%3A%5Ba-zA-Z0-9%5D%28%3F%3A%5Ba-zA-Z0-9._-%5D%5Ba-zA-Z0-9%5D%29%3F%29%7B1%2C63%7D%24&testString=openmeter_good%0Aopenmeter.good%0Aopenmeter-good%0Aopenmeter_bad_%0Aopenmeter-bad-%0Aopenmeter.bad.%0A_openmeter-bad%0A-openmeter-bad%0A.openmeter.bad%0A.openmeter.way_toooooooooooooooooooooooooooooooooooooooooooooooooooooo_long&flags=gm&flavor=pcre2&delimiter=%2F
var keyValueFormat = regexp.MustCompile(`^(?:[a-zA-Z0-9](?:[a-zA-Z0-9._-]?[a-zA-Z0-9])?){1,63}$`)

// https://regex101.com/?regex=%5E%28%3F%3A%5Ba-zA-Z0-9%5D%28%3F%3A%5Ba-zA-Z0-9._-%5D%5Ba-zA-Z0-9%5D%29%3F%29%7B1%2C63%7D%24&testString=openmeter_good%0Aopenmeter.good%0Aopenmeter-good%0Aopenmeter_bad_%0Aopenmeter-bad-%0Aopenmeter.bad.%0A_openmeter-bad%0A-openmeter-bad%0A.openmeter.bad%0A.openmeter.way_toooooooooooooooooooooooooooooooooooooooooooooooooooooo_long&flags=gm&flavor=pcre2&delimiter=%2F
var reservedPrefixMatcher = regexp.MustCompile(`^(_|kong|konnect|insomnia|mesh|kic|kuma|openmeter)`)

func ValidateLabel(k, v string) error {
	var errs []error

	if !keyValueFormat.MatchString(k) {
		errs = append(errs, models.NewValidationIssue(
			"invalid_label_key_format",
			"label key must be a valid DNS_SUBDOMAIN format",
			models.WithFieldString("labels"),
			models.WithCriticalSeverity(),
			commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
		))
	}

	if !keyValueFormat.MatchString(v) {
		errs = append(errs, models.NewValidationIssue(
			"invalid_label_value_format",
			"label value must be a valid DNS_SUBDOMAIN format",
			models.WithFieldString("labels"),
			models.WithCriticalSeverity(),
			commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
		))
	}

	if reservedPrefixMatcher.MatchString(k) {
		errs = append(errs, models.NewValidationIssue(
			"invalid_label_key_prefix",
			"label key must be a valid DNS_SUBDOMAIN format",
			models.WithFieldString("labels"),
			models.WithCriticalSeverity(),
			commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
		))
	}

	return errors.Join(errs...)
}

func ValidateLabels(l api.Labels) error {
	var errs []error

	for k, v := range l {
		if err := ValidateLabel(k, v); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
