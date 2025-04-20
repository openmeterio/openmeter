package plan

import (
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ models.Validator = (*Plan)(nil)

type Plan struct {
	models.NamespacedID
	models.ManagedModel

	productcatalog.PlanMeta

	// Phases
	Phases []Phase `json:"phases"`

	// Addons contains the list of Addons assigned to this plan. It is only provided if the Plan was fetched
	// with Addons being expanded.
	Addons *[]Addon `json:"addons,omitempty"`
}

func (p Plan) Validate() error {
	var errs []error

	if err := p.PlanMeta.Validate(); err != nil {
		errs = append(errs, err)
	}

	for _, phase := range p.Phases {
		if err := phase.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid PlanPhase %q: %s", phase.Name, err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (p Plan) AsProductCatalogPlan() productcatalog.Plan {
	return productcatalog.Plan{
		PlanMeta: p.PlanMeta,
		Phases:   lo.Map(p.Phases, func(phase Phase, _ int) productcatalog.Phase { return phase.AsProductCatalogPhase() }),
	}
}
