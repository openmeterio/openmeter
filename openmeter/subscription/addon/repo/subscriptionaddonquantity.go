package subscriptionaddonrepo

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type subscriptionAddonQuantityRepo struct {
	db *db.Client
}

var _ subscriptionaddon.SubscriptionAddonQuantityRepository = (*subscriptionAddonQuantityRepo)(nil)

func NewSubscriptionAddonQuantityRepo(db *db.Client) *subscriptionAddonQuantityRepo {
	return &subscriptionAddonQuantityRepo{
		db: db,
	}
}

// Create creates a new quantity for a subscription addon
func (r *subscriptionAddonQuantityRepo) Create(ctx context.Context, subscriptionAddonID models.NamespacedID, input subscriptionaddon.CreateSubscriptionAddonQuantityRepositoryInput) (*subscriptionaddon.SubscriptionAddonQuantity, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *subscriptionAddonQuantityRepo) (*subscriptionaddon.SubscriptionAddonQuantity, error) {
		entity, err := repo.db.SubscriptionAddonQuantity.Create().
			SetSubscriptionAddonID(subscriptionAddonID.ID).
			SetActiveFrom(input.ActiveFrom).
			SetQuantity(input.Quantity).
			Save(ctx)
		if err != nil {
			return nil, err
		}

		quantity := MapSubscriptionAddonQuantity(entity)
		return &quantity, nil
	})
}
