package streaming

import (
	"errors"
	"slices"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
)

type QueryParams struct {
	ClientID       *string
	From           *time.Time
	To             *time.Time
	FilterCustomer []Customer
	FilterSubject  []string
	FilterGroupBy  map[string]filter.FilterString
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

	if err := errors.Join(lo.Map(p.FilterCustomer, func(c Customer, _ int) error {
		return c.GetUsageAttribution().Validate()
	})...); err != nil {
		errs = append(errs, err)
	}

	// Validate the group by filters
	for _, filter := range p.FilterGroupBy {
		if err := filter.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return models.NewNillableGenericValidationError(errors.Join(errs...))
	}

	return nil
}

// Customer is a customer that can be used in a meter query
type Customer interface {
	GetUsageAttribution() CustomerUsageAttribution
}

// CustomerUsageAttribution holds customer fields that map usage to a customer
type CustomerUsageAttribution struct {
	ID          string   `json:"id"`
	Key         *string  `json:"key"`
	SubjectKeys []string `json:"subjectKeys"`
}

func (ua CustomerUsageAttribution) Validate() error {
	if ua.ID == "" {
		return models.NewGenericValidationError(errors.New("usage attribution must have an id"))
	}

	if ua.Key == nil && len(ua.SubjectKeys) == 0 {
		return models.NewGenericValidationError(errors.New("usage attribution must have a key or subject keys"))
	}

	return nil
}

// GetValues returns the values by which the usage is attributed to the customer
func (ua CustomerUsageAttribution) GetValues() []string {
	attributions := []string{}

	if ua.Key != nil {
		attributions = append(attributions, *ua.Key)
	}

	attributions = append(attributions, ua.SubjectKeys...)

	return attributions
}

func (ua CustomerUsageAttribution) Equal(other CustomerUsageAttribution) bool {
	return ua.ID == other.ID && ua.Key == other.Key && slices.Equal(ua.SubjectKeys, other.SubjectKeys)
}
