package addon

import (
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

func IsAddonDeleted(at time.Time) models.ValidatorFunc[Addon] {
	return func(a Addon) error {
		if a.IsDeleted() {
			return fmt.Errorf("add-on is deleted [deleted_at=%s]", *a.DeletedAt)
		}

		return nil
	}
}

func HasAddonStatus(statuses ...productcatalog.AddonStatus) models.ValidatorFunc[Addon] {
	return func(a Addon) error {
		if !lo.Contains(statuses, a.Status()) {
			return fmt.Errorf("invalid %s status, allowed statuses: %+v", a.Status(), statuses)
		}

		return nil
	}
}
