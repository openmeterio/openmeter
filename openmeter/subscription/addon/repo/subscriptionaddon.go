package subscriptionaddonrepo

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	addondb "github.com/openmeterio/openmeter/openmeter/ent/db/addon"
	dbsubscriptionaddon "github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionaddon"
	dbsubscriptionaddonquantity "github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionaddonquantity"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type subscriptionAddonRepo struct {
	db *db.Client
}

var _ subscriptionaddon.SubscriptionAddonRepository = (*subscriptionAddonRepo)(nil)

func NewSubscriptionAddonRepo(db *db.Client) *subscriptionAddonRepo {
	return &subscriptionAddonRepo{
		db: db,
	}
}

// Create creates a new subscription addon
func (r *subscriptionAddonRepo) Create(ctx context.Context, namespace string, input subscriptionaddon.CreateSubscriptionAddonRepositoryInput) (*models.NamespacedID, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *subscriptionAddonRepo) (*models.NamespacedID, error) {
		cmd := repo.db.SubscriptionAddon.Create().
			SetNamespace(namespace).
			SetAddonID(input.AddonID).
			SetSubscriptionID(input.SubscriptionID)

		if input.Metadata != nil {
			cmd = cmd.SetMetadata(input.Metadata)
		}

		entity, err := cmd.Save(ctx)
		if err != nil {
			return nil, err
		}

		return &models.NamespacedID{
			ID:        entity.ID,
			Namespace: entity.Namespace,
		}, nil
	})
}

// Get retrieves a subscription addon by ID
func (r *subscriptionAddonRepo) Get(ctx context.Context, params subscriptionaddon.GetSubscriptionAddonInput) (*subscriptionaddon.SubscriptionAddon, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *subscriptionAddonRepo) (*subscriptionaddon.SubscriptionAddon, error) {
		query := querySubscriptionAddon(repo.db.SubscriptionAddon.Query())

		if params.ID != "" {
			query = query.Where(
				dbsubscriptionaddon.ID(params.ID),
				dbsubscriptionaddon.Namespace(params.Namespace),
			)
			if params.SubscriptionID != "" {
				query = query.Where(dbsubscriptionaddon.SubscriptionID(params.SubscriptionID))
			}
		} else {
			query = query.Where(
				dbsubscriptionaddon.Namespace(params.Namespace),
				dbsubscriptionaddon.SubscriptionID(params.SubscriptionID),
				dbsubscriptionaddon.HasAddonWith(addondb.Or(addondb.ID(params.AddonIDOrKey), addondb.Key(params.AddonIDOrKey))),
			)
		}

		entity, err := query.Only(ctx)
		if err != nil {
			if db.IsNotFound(err) {
				return nil, models.NewGenericNotFoundError(
					fmt.Errorf("subscription addon %s not found", params.ID),
				)
			}

			return nil, err
		}

		addon, err := MapSubscriptionAddon(entity)
		if err != nil {
			return nil, err
		}

		return &addon, nil
	})
}

// List retrieves multiple subscription addons
func (r *subscriptionAddonRepo) List(ctx context.Context, namespace string, filter subscriptionaddon.ListSubscriptionAddonRepositoryInput) (pagination.Result[subscriptionaddon.SubscriptionAddon], error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *subscriptionAddonRepo) (pagination.Result[subscriptionaddon.SubscriptionAddon], error) {
		query := querySubscriptionAddon(repo.db.SubscriptionAddon.Query()).
			Where(
				dbsubscriptionaddon.Namespace(namespace),
				dbsubscriptionaddon.SubscriptionID(filter.SubscriptionID),
			)

		order := entutils.GetOrdering(sortx.OrderAsc)
		if !filter.Order.IsDefaultValue() {
			order = entutils.GetOrdering(filter.Order)
		}

		switch filter.OrderBy {
		case subscriptionaddon.OrderByID:
			query = query.Order(dbsubscriptionaddon.ByID(order...))
		case subscriptionaddon.OrderByUpdatedAt:
			query = query.Order(dbsubscriptionaddon.ByUpdatedAt(order...))
		case subscriptionaddon.OrderByCreatedAt:
			fallthrough
		default:
			query = query.Order(dbsubscriptionaddon.ByCreatedAt(order...))
		}

		// Let's return everything if no pagination is requested
		if filter.Page.IsZero() {
			entities, err := query.All(ctx)
			if err != nil {
				return pagination.Result[subscriptionaddon.SubscriptionAddon]{}, err
			}

			items, err := MapSubscriptionAddons(entities)
			if err != nil {
				return pagination.Result[subscriptionaddon.SubscriptionAddon]{}, err
			}
			return pagination.Result[subscriptionaddon.SubscriptionAddon]{
				Items:      items,
				Page:       pagination.NewPage(1, len(items)),
				TotalCount: len(items),
			}, nil
		}

		paged, err := query.Paginate(ctx, filter.Page)
		if err != nil {
			return pagination.Result[subscriptionaddon.SubscriptionAddon]{}, err
		}

		return entutils.MapPagedWithErr(paged, MapSubscriptionAddon)
	})
}

func querySubscriptionAddon(query *db.SubscriptionAddonQuery) *db.SubscriptionAddonQuery {
	return query.
		WithAddon(func(aq *db.AddonQuery) {
			aq.WithRatecards(func(arq *db.AddonRateCardQuery) {
				arq.WithFeatures()
				arq.WithTaxCode()
			})
		}).
		WithQuantities(func(saqq *db.SubscriptionAddonQuantityQuery) {
			saqq.Order(
				db.Asc(dbsubscriptionaddonquantity.FieldActiveFrom),
				db.Asc(dbsubscriptionaddonquantity.FieldCreatedAt),
			)
		})
}
