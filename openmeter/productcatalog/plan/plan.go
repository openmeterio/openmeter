package plan

import (
	"errors"
	"fmt"
	"time"

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

	return productcatalog.NewValidationError(errors.Join(errs...))
}

func (p Plan) AsProductCatalogPlan(at time.Time) (productcatalog.Plan, error) {
	// We filter out deleted resources. Its an interesting mind-bender why we'd have deleted resources in the first place...
	// Let's start with the plan itself
	if p.DeletedAt != nil && !at.Before(*p.DeletedAt) {
		return productcatalog.Plan{}, errors.New("plan is deleted")
	}

	// Then continue with the phases
	phases := lo.Filter(p.Phases, func(phase Phase, _ int) bool {
		return phase.DeletedAt == nil || at.After(*phase.DeletedAt)
	})

	return productcatalog.Plan{
		PlanMeta: p.PlanMeta,
		Phases:   lo.Map(phases, func(phase Phase, _ int) productcatalog.Phase { return phase.AsProductCatalogPhase() }),
	}, nil
}
