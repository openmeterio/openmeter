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

	// If provided, cannot be an empty string
	if p.ClientID != nil && len(*p.ClientID) == 0 {
		errs = append(errs, errors.New("client id cannot be empty"))
	}

	// Check that from and to are consistent
	if p.From != nil && p.To != nil {
		if p.From.Equal(*p.To) {
			errs = append(errs, errors.New("from and to cannot be equal"))
		}

		if p.From.After(*p.To) {
			errs = append(errs, errors.New("from must be before to"))
		}
	}

	// This is required because otherwise the response would be ambiguous
	if len(p.FilterSubject) > 1 && !slices.Contains(p.GroupBy, "subject") {
		errs = append(errs, errors.New("multiple subject filters are only allowed with subject group by"))
	}

	// This is required because otherwise the response would be ambiguous
	if len(p.FilterCustomer) > 1 && !slices.Contains(p.GroupBy, "customer_id") {
		errs = append(errs, errors.New("multiple customer filters are only allowed with customer_id group by"))
	}

	// This is required for now because we don't support customer_id without a filter
	// To support this we need to map all subjects to customer_ids
	if slices.Contains(p.GroupBy, "customer_id") && len(p.FilterCustomer) == 0 {
		errs = append(errs, errors.New("customer filter is required with customer_id group by"))
	}

	if len(errs) > 0 {
		return models.NewNillableGenericValidationError(errors.Join(errs...))
	}

	return nil
}
