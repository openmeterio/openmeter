package charges

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

type WithIndex[T any] struct {
	Index int
	Value T
}

// ValidateStandardInvoiceCreatedFeatures reports missing feature references as
// line-scoped validation issues before a charge engine performs lifecycle side effects.
func ValidateStandardInvoiceCreatedFeatures(input billing.OnStandardInvoiceCreatedInput) error {
	var errs []error

	for _, line := range input.Lines {
		if line.GetFeatureMeterRef() == nil || input.FeatureMeters.Has(line) {
			continue
		}

		_, err := input.FeatureMeters.Get(line)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
