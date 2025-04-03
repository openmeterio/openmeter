package subscriptionaddonrepo

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

type subscriptionAddonRateCardRepo struct {
	db *db.Client
}

var _ subscriptionaddon.SubscriptionAddonRateCardRepository = (*subscriptionAddonRateCardRepo)(nil)

func NewSubscriptionAddonRateCardRepo(db *db.Client) *subscriptionAddonRateCardRepo {
	return &subscriptionAddonRateCardRepo{
		db: db,
	}
}

// CreateMany creates multiple rate cards for a subscription addon
func (r *subscriptionAddonRateCardRepo) CreateMany(ctx context.Context, subscriptionAddonID models.NamespacedID, inputs []subscriptionaddon.CreateSubscriptionAddonRateCardRepositoryInput) ([]subscriptionaddon.SubscriptionAddonRateCard, error) {
	return transaction.Run(ctx, r, func(ctx context.Context) ([]subscriptionaddon.SubscriptionAddonRateCard, error) {
		return entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *subscriptionAddonRateCardRepo) ([]subscriptionaddon.SubscriptionAddonRateCard, error) {
			var results []subscriptionaddon.SubscriptionAddonRateCard

			// For each input, create a rate card and its links
			for _, input := range inputs {
				// Create the rate card
				rateCard, err := repo.db.SubscriptionAddonRateCard.Create().
					SetSubscriptionAddonID(subscriptionAddonID.ID).
					Save(ctx)
				if err != nil {
					return nil, err
				}

				// Create links to subscription items for this rate card
				links, err := repo.db.SubscriptionAddonRateCardItemLink.CreateBulk(
					lo.Map(input.AffectedSubscriptionItemIDs, func(itemID string, _ int) *db.SubscriptionAddonRateCardItemLinkCreate {
						return repo.db.SubscriptionAddonRateCardItemLink.Create().
							SetSubscriptionAddonRateCardID(rateCard.ID).
							SetSubscriptionItemID(itemID)
					})...,
				).Save(ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to create links to subscription items: %w", err)
				}

				rateCard.Edges.Items = links

				// Map to domain model
				result := MapSubscriptionAddonRateCard(rateCard)

				results = append(results, result)
			}

			return results, nil
		})
	})
}
