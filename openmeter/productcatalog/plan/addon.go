package plan

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ models.Validator = (*Addon)(nil)

// Addon stores the Plan specific representation of planaddon.PlanAddon.
type Addon struct {
	models.NamespacedID
	models.ManagedModel

	productcatalog.PlanAddonMeta
	productcatalog.Addon
}

func (a Addon) Validate() error {
	var errs []error

	if a.Namespace == "" {
		errs = append(errs, productcatalog.ErrNamespaceEmpty)
	}

	if a.ID == "" {
		errs = append(errs, productcatalog.ErrIDEmpty)
	}

	if err := a.Addon.Validate(); err != nil {
		errs = append(errs, err)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
