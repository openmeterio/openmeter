package addon

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ models.Validator = (*Addon)(nil)

type Addon struct {
	models.NamespacedID
	models.ManagedModel

	productcatalog.AddonMeta

	// RateCards
	RateCards RateCards `json:"rateCards"`
}

func (a Addon) Validate() error {
	var errs []error

	if err := a.NamespacedID.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := a.ManagedModel.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := a.AddonMeta.Validate(); err != nil {
		errs = append(errs, err)
	}

	for _, rateCard := range a.RateCards {
		if err := rateCard.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
