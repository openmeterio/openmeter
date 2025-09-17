package subscription

import (
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type ListSubscriptionsInput struct {
	pagination.Page

	Namespaces     []string
	Customers      []string
	ActiveAt       *time.Time
	ActiveInPeriod *timeutil.ClosedPeriod
}

func (i ListSubscriptionsInput) Validate() error {
	var errs []error

	if i.ActiveInPeriod != nil {
		if err := i.ActiveInPeriod.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("active in period: %w", err))
		}
	}

	return errors.Join(errs...)
}

type SubscriptionList = pagination.Result[Subscription]
