package subscription

import (
	"time"

	"github.com/openmeterio/openmeter/pkg/pagination"
)

type ListSubscriptionsInput struct {
	pagination.Page

	Namespaces []string
	Customers  []string
	ActiveAt   *time.Time
}

func (i ListSubscriptionsInput) Validate() error {
	return nil
}

type SubscriptionList = pagination.PagedResponse[Subscription]
