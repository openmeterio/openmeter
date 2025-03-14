package subscription

import (
	"errors"
	"time"

	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type ListSubscriptionsInput struct {
	pagination.Page

	// If set to true, all subscriptions will be expanded by the service to their full view in process, making the response larger and slower
	ExpandToView bool

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

type (
	PagedSubscriptions     = pagination.PagedResponse[Subscription]
	PagedSubscriptionViews = pagination.PagedResponse[SubscriptionView]
	SubscriptionList       = mo.Either[PagedSubscriptions, PagedSubscriptionViews]
)
