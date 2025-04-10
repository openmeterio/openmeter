package subscriptionaddonrepo

import (
	"context"
	"errors"
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
					SetNamespace(subscriptionAddonID.Namespace).
					SetSubscriptionAddonID(subscriptionAddonID.ID).
					SetAddonRatecardID(input.AddonRateCardID).
					Save(ctx)
				if err != nil {
					return nil, err
				}

				// Create links to subscription items for this rate card
				links, err := repo.db.SubscriptionAddonRateCardItemLink.CreateBulk(
					lo.Map(input.AffectedSubscriptionItems, func(item subscriptionaddon.SubscriptionAddonRateCardItemRef, _ int) *db.SubscriptionAddonRateCardItemLinkCreate {
						return repo.db.SubscriptionAddonRateCardItemLink.Create().
							SetSubscriptionAddonRateCardID(rateCard.ID).
							SetSubscriptionItemID(item.SubscriptionItemID).
							SetSubscriptionItemThroughID(item.SubscriptionItemThroughID)
					})...,
				).Save(ctx)
				if err != nil {
					// For magical reasons, the constraint error looks like this:
					// failed to create subscription addon rate cards: failed to create links to subscription items: db: constraint failed: insert nodes to table \"subscription_addon_rate_card_item_links\": ERROR: insert or update on table \"subscription_addon_rate_card_item_links\" violates foreign key constraint \"subscription_addon_rate_card_i_5443b55d7e58df21cd89a6726b500989\" (SQLSTATE 23503)"
					// So we cannot assert for the specific type of constraint we're violating
					if db.IsConstraintError(err) {
						return nil, models.NewGenericNotFoundError(errors.New("constraint failed, resource not found"))
					}

					return nil, fmt.Errorf("failed to create links to subscription items: %w", err)
				}

				rateCard.Edges.Items = links

				// Map to domain model
				result, err := MapSubscriptionAddonRateCard(rateCard)
				if err != nil {
					return nil, fmt.Errorf("failed to map rate card to domain model: %w", err)
				}

				results = append(results, result)
			}

			return results, nil
		})
	})
}
