package subscriptionaddon

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

type CreateSubscriptionAddonRepositoryInput struct {
	models.MetadataModel

	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`

	AddonID        string `json:"addonID"`
	SubscriptionID string `json:"subscriptionID"`
}

type SubscriptionAddonRepository interface {
	Create(ctx context.Context, namespace string, input CreateSubscriptionAddonRepositoryInput) (*SubscriptionAddon, error)
}

type CreateSubscriptionAddonRateCardRepositoryInput struct {
	AffectedSubscriptionItemIDs []string `json:"affectedSubscriptionItemIDs"`
}

type SubscriptionAddonRateCardRepository interface {
	CreateMany(ctx context.Context, subscriptionAddonID models.NamespacedID, inputs []CreateSubscriptionAddonRateCardRepositoryInput) (*SubscriptionAddonRateCard, error)
}

type CreateSubscriptionAddonQuantityRepositoryInput struct {
	ActiveFrom time.Time `json:"activeFrom"`
	Quantity   int       `json:"quantity"`
}

type SubscriptionAddonQuantityRepository interface {
	Create(ctx context.Context, subscriptionAddonID models.NamespacedID, input CreateSubscriptionAddonQuantityRepositoryInput) (*SubscriptionAddonQuantity, error)
}
