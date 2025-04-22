package addon

import (
	"errors"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ models.Validator = (*Addon)(nil)

// Plan stores the Addon specific representation of planaddon.PlanAddon.
type Plan struct {
	models.NamespacedID
	models.ManagedModel

	productcatalog.PlanAddonMeta
	productcatalog.Plan
}

func (p Plan) Validate() error {
	var errs []error

	if err := p.NamespacedID.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := p.Plan.Validate(); err != nil {
		errs = append(errs, err)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
