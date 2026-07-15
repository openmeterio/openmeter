package creditpurchase

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type FeatureFilters []string

func (f FeatureFilters) Validate() error {
	var errs []error

	for i, key := range f {
		if key == "" {
			errs = append(errs, fmt.Errorf("[%d]: feature key is required", i))
		}
	}

	if len(f.Normalize()) != len(f) {
		errs = append(errs, errors.New("duplicate feature key"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (f FeatureFilters) Normalize() FeatureFilters {
	return FeatureFilters(slicesx.Normalize([]string(f)))
}

// ValidateAsFeatureFilter validates the customer-facing filter form: one or
// more features matched with any-of semantics. An empty restricted filter is
// rejected — the all-features and unrestricted-only cases have their own
// filter forms.
func (f FeatureFilters) ValidateAsFeatureFilter() error {
	if len(f) == 0 {
		return errors.New("features are required when feature filter is restricted")
	}

	if err := f.Validate(); err != nil {
		return fmt.Errorf("features: %w", err)
	}

	return nil
}
