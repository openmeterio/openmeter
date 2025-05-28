package plan

import (
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

// TODO(chrisgacsal): rename to ValidatePlanNotDeleted to be alined wit the rest of the validators in productcatalog package.
func IsPlanDeleted(at time.Time) models.ValidatorFunc[Plan] {
	return func(p Plan) error {
		if p.IsDeletedAt(at) {
			return fmt.Errorf("plan is deleted [deletedAt=%s]", *p.DeletedAt)
		}

		return nil
	}
}

// TODO(chrisgacsal): rename to ValidatePlanWithStatus to be alined wit the rest of the validators in productcatalog package.
func HasPlanStatus(statuses ...productcatalog.PlanStatus) models.ValidatorFunc[Plan] {
	return func(p Plan) error {
		if !lo.Contains(statuses, p.Status()) {
			return fmt.Errorf("invalid %s status, allowed statuses: %+v", p.Status(), statuses)
		}

		return nil
	}
}
