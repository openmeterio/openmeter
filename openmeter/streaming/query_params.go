package streaming

import (
	"errors"
	"time"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
)

type QueryParams struct {
	ClientID       *string
	From           *time.Time
	To             *time.Time
	FilterSubject  []string
	FilterGroupBy  map[string][]string
	GroupBy        []string
	WindowSize     *meter.WindowSize
	WindowTimeZone *time.Location
}

type QueryParamsV2 struct {
	ClientID       *string
	GroupBy        []string
	WindowSize     *meter.WindowSize
	WindowTimeZone *time.Location
	Filter         *Filter
}

func (p *QueryParamsV2) Validate() error {
	var errs []error

	if p.ClientID != nil && len(*p.ClientID) == 0 {
		errs = append(errs, errors.New("client id cannot be empty"))
	}

	if err := p.Filter.Validate(); err != nil {
		errs = append(errs, err)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type Filter struct {
	GroupBy *map[string]filter.FilterString
	Subject *filter.FilterString
	Time    *filter.FilterTime
}

func (f *Filter) Validate() error {
	var errs []error

	if f == nil {
		return nil
	}

	if f.GroupBy != nil {
		for _, v := range *f.GroupBy {
			if err := v.ValidateWithComplexity(1); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if f.Subject != nil {
		if err := f.Subject.ValidateWithComplexity(1); err != nil {
			errs = append(errs, err)
		}
	}

	if f.Time != nil {
		if err := f.Time.ValidateWithComplexity(1); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// Validate validates query params focusing on `from` and `to` being aligned with query and meter window sizes
func (p *QueryParams) Validate() error {
	var errs []error

	if p.ClientID != nil && len(*p.ClientID) == 0 {
		errs = append(errs, errors.New("client id cannot be empty"))
	}

	if p.From != nil && p.To != nil {
		if p.From.Equal(*p.To) {
			errs = append(errs, errors.New("from and to cannot be equal"))
		}

		if p.From.After(*p.To) {
			errs = append(errs, errors.New("from must be before to"))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
