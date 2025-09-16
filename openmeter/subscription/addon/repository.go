package subscriptionaddon

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// SubscriptionAddon
type CreateSubscriptionAddonRepositoryInput struct {
	models.MetadataModel

	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`

	AddonID        string `json:"addonID"`
	SubscriptionID string `json:"subscriptionID"`
}

type ListSubscriptionAddonRepositoryInput struct {
	SubscriptionID string `json:"subscriptionID"`

	pagination.Page
}

type SubscriptionAddonRepository interface {
	Create(ctx context.Context, namespace string, input CreateSubscriptionAddonRepositoryInput) (*models.NamespacedID, error)
	Get(ctx context.Context, id models.NamespacedID) (*SubscriptionAddon, error)
	List(ctx context.Context, namespace string, filter ListSubscriptionAddonRepositoryInput) (pagination.Result[SubscriptionAddon], error)
}

// SubscriptionAddonQuantity
type CreateSubscriptionAddonQuantityRepositoryInput struct {
	ActiveFrom time.Time `json:"activeFrom"`
	Quantity   int       `json:"quantity"`
}

type SubscriptionAddonQuantityRepository interface {
	Create(ctx context.Context, subscriptionAddonID models.NamespacedID, input CreateSubscriptionAddonQuantityRepositoryInput) (*SubscriptionAddonQuantity, error)
}
