package streaming

import (
	"errors"
	"slices"
	"time"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/models"
)

type QueryParams struct {
	ClientID       *string
	From           *time.Time
	To             *time.Time
	FilterCustomer []customer.Customer
	FilterSubject  []string
	FilterGroupBy  map[string][]string
	GroupBy        []string
	WindowSize     *meter.WindowSize
	WindowTimeZone *time.Location
}

// Validate validates query params focusing on `from` and `to` being aligned with query and meter window sizes
func (p *QueryParams) Validate() error {
	var errs []error

	if p.ClientID != nil && len(*p.ClientID) == 0 {
		return errors.New("client id cannot be empty")
	}

	if p.From != nil && p.To != nil {
		if p.From.Equal(*p.To) {
			errs = append(errs, errors.New("from and to cannot be equal"))
		}

		if p.From.After(*p.To) {
			errs = append(errs, errors.New("from must be before to"))
		}
	}

	if len(p.FilterCustomer) > 0 && len(p.FilterSubject) > 0 {
		errs = append(errs, errors.New("filter customer and filter subject cannot be used together"))
	}

	if slices.Contains(p.GroupBy, "customer_id") && len(p.FilterCustomer) == 0 {
		errs = append(errs, errors.New("filter customer is required when grouping by customer_id"))
	}

	if len(errs) > 0 {
		return models.NewNillableGenericValidationError(errors.Join(errs...))
	}

	return nil
}
