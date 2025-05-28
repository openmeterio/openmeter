package plan

import (
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

var (
	_ models.Validator             = (*Plan)(nil)
	_ models.CustomValidator[Plan] = (*Plan)(nil)
)

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

func (p Plan) ValidateWith(validators ...models.ValidatorFunc[Plan]) error {
	return models.Validate(p, validators...)
}

func (p Plan) Validate() error {
	return p.ValidateWith(
		ValidatePlanMeta(),
		ValidatePlanPhases(),
	)
}

func (p Plan) AsProductCatalogPlan() productcatalog.Plan {
	return productcatalog.Plan{
		PlanMeta: p.PlanMeta,
		Phases:   lo.Map(p.Phases, func(phase Phase, _ int) productcatalog.Phase { return phase.AsProductCatalogPhase() }),
	}
}

func ValidatePlanMeta() models.ValidatorFunc[Plan] {
	return func(p Plan) error {
		return p.PlanMeta.Validate()
	}
}

func ValidatePlanPhases() models.ValidatorFunc[Plan] {
	return func(p Plan) error {
		var errs []error

		for _, phase := range p.Phases {
			if err := phase.Validate(); err != nil {
				errs = append(errs, fmt.Errorf("invalid plan phase %q: %s", phase.Key, err))
			}
		}

		return models.NewNillableGenericValidationError(errors.Join(errs...))
	}
}
