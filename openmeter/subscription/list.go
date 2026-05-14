package subscription

import (
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type OrderBy string

const (
	OrderByID         OrderBy = "id"
	OrderByActiveFrom OrderBy = "activeFrom"
	OrderByActiveTo   OrderBy = "activeTo"
)

func (o OrderBy) Validate() error {
	switch o {
	case OrderByID, OrderByActiveFrom, OrderByActiveTo:
		return nil
	}
	return fmt.Errorf("invalid order by: %s", o)
}

type ListSubscriptionsInput struct {
	pagination.Page
	OrderBy OrderBy
	Order   sortx.Order

	Namespaces     []string
	CustomerID     *filter.FilterULID
	ActiveAt       *time.Time
	ActiveInPeriod *timeutil.StartBoundedPeriod
	Status         []SubscriptionStatus

	ID      *filter.FilterULID
	PlanID  *filter.FilterULID
	PlanKey *filter.FilterString
}

func (i ListSubscriptionsInput) Validate() error {
	var errs []error

	if i.ActiveInPeriod != nil {
		if err := i.ActiveInPeriod.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("active in period: %w", err))
		}
	}

	if i.OrderBy != "" {
		if err := i.OrderBy.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("order by: %w", err))
		}
	}

	if !i.Page.IsZero() {
		if err := i.Page.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("page: %w", err))
		}
	}

	if i.ID != nil {
		if err := i.ID.Validate(); err != nil {
			errs = append(errs, models.NewGenericValidationError(fmt.Errorf("invalid id filter: %w", err)))
		}
	}

	if i.CustomerID != nil {
		if err := i.CustomerID.Validate(); err != nil {
			errs = append(errs, models.NewGenericValidationError(fmt.Errorf("invalid customer_id filter: %w", err)))
		}
	}

	if i.PlanID != nil {
		if err := i.PlanID.Validate(); err != nil {
			errs = append(errs, models.NewGenericValidationError(fmt.Errorf("invalid plan_id filter: %w", err)))
		}
	}

	if i.PlanKey != nil {
		if err := i.PlanKey.Validate(); err != nil {
			errs = append(errs, models.NewGenericValidationError(fmt.Errorf("invalid plan_key filter: %w", err)))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type SubscriptionList = pagination.Result[Subscription]
