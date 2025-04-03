package subscriptionaddon

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type ListSubscriptionAddonsInput struct {
	SubscriptionID string `json:"subscriptionID"`
}

type Service interface {
	Create(ctx context.Context, namespace string, input CreateSubscriptionAddonInput) (*SubscriptionAddon, error)
	Get(ctx context.Context, id models.NamespacedID) (*SubscriptionAddon, error)
	List(ctx context.Context, namespace string, input ListSubscriptionAddonsInput) (pagination.PagedResponse[SubscriptionAddon], error)

	ChangeQuantity(ctx context.Context, id models.NamespacedID, input CreateSubscriptionAddonQuantityInput) (*SubscriptionAddon, error)
}
