package creditpurchase

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/models"
)

// FeatureKeyFilter filters charges by their feature_filters restriction.
// Unlike the balance-side coverage filter, unrestricted charges do not match a
// keyed filter: a grant usable by any feature is not "scoped to" the requested
// feature. Exactly one of In or Exists may be set; the zero value matches
// everything.
type FeatureKeyFilter struct {
	// In matches charges whose feature restriction includes any of these keys.
	In []string

	// Exists, when set, matches charges purely by whether they carry a feature
	// restriction at all: true selects restricted charges, false unrestricted
	// ones.
	Exists *bool
}

func (f FeatureKeyFilter) Validate() error {
	var errs []error

	if f.Exists != nil && len(f.In) > 0 {
		errs = append(errs, errors.New("exists and key filters cannot both be set"))
	}

	for i, key := range f.In {
		if key == "" {
			errs = append(errs, fmt.Errorf("in[%d]: feature key is required", i))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
