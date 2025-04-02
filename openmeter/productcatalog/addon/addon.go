package addon

import (
	"errors"
	"time"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ models.Validator = (*Addon)(nil)

type Addon struct {
	models.NamespacedID
	models.ManagedModel

	productcatalog.AddonMeta

	// RateCards
	RateCards productcatalog.RateCards `json:"rateCards"`
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

	if err := a.RateCards.Validate(); err != nil {
		errs = append(errs, err)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (a Addon) AsProductCatalogAddon(at time.Time) (productcatalog.Addon, error) {
	if a.DeletedAt != nil && !at.Before(*a.DeletedAt) {
		return productcatalog.Addon{}, errors.New("add-on is deleted")
	}

	return productcatalog.Addon{
		AddonMeta: a.AddonMeta,
		RateCards: a.RateCards,
	}, nil
}
