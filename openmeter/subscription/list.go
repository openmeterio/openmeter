package subscription

import (
	"errors"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type ListSubscriptionsInput struct {
	pagination.Page

	Namespaces []string
	Customers  []string
	ActiveAt   *time.Time
}

func (i ListSubscriptionsInput) Validate() error {
	if len(i.Namespaces) == 0 {
		return models.NewGenericValidationError(errors.New("namespace is required"))
	}

	return nil
}

type SubscriptionList = pagination.PagedResponse[Subscription]
