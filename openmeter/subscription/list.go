package subscription

import (
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type OrderBy string

const (
	OrderByActiveFrom OrderBy = "activeFrom"
	OrderByActiveTo   OrderBy = "activeTo"
)

func (o OrderBy) Validate() error {
	switch o {
	case OrderByActiveFrom, OrderByActiveTo:
		return nil
	}
	return fmt.Errorf("invalid order by: %s", o)
}

type ListSubscriptionsInput struct {
	pagination.Page
	OrderBy OrderBy
	Order   sortx.Order

	Namespaces     []string
	CustomerIDs    []string
	ActiveAt       *time.Time
	ActiveInPeriod *timeutil.StartBoundedPeriod
	Status         []SubscriptionStatus
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

	if len(i.Status) > 0 {
		for _, status := range i.Status {
			if err := status.Validate(); err != nil {
				errs = append(errs, fmt.Errorf("status: %w", err))
			}
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type SubscriptionList = pagination.Result[Subscription]
