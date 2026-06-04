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

	if len(f.Strings()) != len(f) {
		errs = append(errs, errors.New("duplicate feature key"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (f FeatureFilters) Normalize() FeatureFilters {
	return FeatureFilters(slicesx.Normalize([]string(f)))
}

func (f FeatureFilters) Strings() []string {
	return []string(f.Normalize())
}
