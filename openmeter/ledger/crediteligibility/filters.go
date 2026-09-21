// Package crediteligibility matches credit filters against ledger routes.
package crediteligibility

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

// Filters stores credit restrictions. Empty features impose no restriction.
type Filters struct {
	Features []string `json:"features,omitempty"`
}

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

// ValidateAsFeatureFilter validates the singular customer-facing filter form.
// Credit routes may be restricted to multiple features, but a spendability
// query can only ask for one feature at a time.
func (f FeatureFilters) ValidateAsFeatureFilter() error {
	switch len(f) {
	case 0:
		return errors.New("features are required when feature filter is restricted")
	case 1:
	default:
		return errors.New("feature filter supports exactly one feature")
	}

	if err := f.Validate(); err != nil {
		return fmt.Errorf("features: %w", err)
	}

	return nil
}

func (f Filters) Validate() error { return FeatureFilters(f.Features).Validate() }
func (f Filters) Normalize() Filters {
	return Filters{Features: FeatureFilters(f.Features).Normalize()}
}
