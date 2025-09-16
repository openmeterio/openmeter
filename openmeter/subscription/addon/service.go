package subscriptionaddon

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Service interface {
	Create(ctx context.Context, namespace string, input CreateSubscriptionAddonInput) (*SubscriptionAddon, error)
	Get(ctx context.Context, id models.NamespacedID) (*SubscriptionAddon, error)
	List(ctx context.Context, namespace string, input ListSubscriptionAddonsInput) (pagination.Result[SubscriptionAddon], error)

	ChangeQuantity(ctx context.Context, id models.NamespacedID, input CreateSubscriptionAddonQuantityInput) (*SubscriptionAddon, error)
}

type ListSubscriptionAddonsInput struct {
	SubscriptionID string `json:"subscriptionID"`

	pagination.Page
}

func (i ListSubscriptionAddonsInput) Validate() error {
	var errs []error

	if i.SubscriptionID == "" {
		errs = append(errs, errors.New("filter has to be provided, all values are empty"))
	}

	return errors.Join(errs...)
}
