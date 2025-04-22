package plan

import (
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

func IsPlanDeleted(at time.Time) models.ValidatorFunc[Plan] {
	return func(p Plan) error {
		if p.IsDeleted() {
			return fmt.Errorf("plan is deleted [deleted_at=%s]", *p.DeletedAt)
		}

		return nil
	}
}

func HasPlanStatus(statuses ...productcatalog.PlanStatus) models.ValidatorFunc[Plan] {
	return func(p Plan) error {
		if !lo.Contains(statuses, p.Status()) {
			return fmt.Errorf("invalid %s status, allowed statuses: %+v", p.Status(), statuses)
		}

		return nil
	}
}
