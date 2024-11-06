package price

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	dbprice "github.com/openmeterio/openmeter/openmeter/ent/db/price"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

type GetPriceFilters struct {
	IncludeDeleted bool
}

type Repository interface {
	models.CadencedResourceRepo[Price]

	GetForSubscription(ctx context.Context, subscriptionId models.NamespacedID, filters GetPriceFilters) ([]Price, error)
	Create(ctx context.Context, input CreateInput) (*Price, error)
	Delete(ctx context.Context, id models.NamespacedID) error
}

type repository struct {
	db *db.Client
}

var _ Repository = &repository{}

func NewRepository(db *db.Client) *repository {
	return &repository{db: db}
}

func (r *repository) Delete(ctx context.Context, id models.NamespacedID) error {
	_, err := entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *repository) (interface{}, error) {
		_, err := repo.db.Price.UpdateOneID(id.ID).
			SetDeletedAt(clock.Now()).
			Save(ctx)
		if db.IsNotFound(err) {
			return nil, NotFoundError{ID: id.ID}
		}
		if err != nil {
			return nil, fmt.Errorf("failed to delete price: %w", err)
		}
		return nil, nil
	})
	return err
}

func (r *repository) GetForSubscription(ctx context.Context, subscriptionId models.NamespacedID, filters GetPriceFilters) ([]Price, error) {
	return entutils.TransactingRepo(
		ctx,
		r,
		func(ctx context.Context, repo *repository) ([]Price, error) {
			query := repo.db.Price.Query().
				Where(
					dbprice.Namespace(subscriptionId.Namespace),
					dbprice.SubscriptionID(subscriptionId.ID),
				)

			if !filters.IncludeDeleted {
				query = query.Where(dbprice.Or(dbprice.DeletedAtIsNil(), dbprice.DeletedAtGT(clock.Now())))
			}

			entities, err := query.All(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get prices for subscription %s: %w", subscriptionId, err)
			}

			prices := make([]Price, len(entities))
			for i, e := range entities {
				prices[i], err = mapDBPriceToPrice(e)
				if err != nil {
					return nil, fmt.Errorf("failed to map price %s: %w", e.ID, err)
				}
			}

			return prices, nil
		},
	)
}

func (r *repository) Create(ctx context.Context, input CreateInput) (*Price, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *repository) (*Price, error) {
		entity, err := repo.db.Price.Create().
			SetNamespace(input.SubscriptionId.Namespace).
			SetSubscriptionID(input.SubscriptionId.ID).
			SetPhaseKey(input.PhaseKey).
			SetItemKey(input.ItemKey).
			SetValue(&input.Value).
			SetKey(input.Key).
			SetActiveFrom(input.ActiveFrom.UTC()).
			SetNillableActiveTo(convert.SafeToUTC(input.ActiveTo)).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create price: %w", err)
		}

		price, err := mapDBPriceToPrice(entity)
		if err != nil {
			return nil, fmt.Errorf("failed to map price: %w", err)
		}

		return lo.ToPtr(price), nil
	})
}

func (r *repository) EndCadence(ctx context.Context, id string, at *time.Time) (*Price, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *repository) (*Price, error) {
		entity, err := repo.db.Price.UpdateOneID(id).
			SetOrClearActiveTo(at).
			Save(ctx)

		if db.IsNotFound(err) {
			return nil, NotFoundError{ID: id}
		}

		if err != nil {
			return nil, fmt.Errorf("failed to end price: %w", err)
		}

		price, err := mapDBPriceToPrice(entity)
		if err != nil {
			return nil, fmt.Errorf("failed to map price: %w", err)
		}

		return lo.ToPtr(price), nil
	})
}

func (r *repository) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	txCtx, rawConfig, eDriver, err := r.db.HijackTx(ctx, &sql.TxOptions{
		ReadOnly: false,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
	}
	return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil
}

func (r *repository) WithTx(ctx context.Context, tx *entutils.TxDriver) *repository {
	txClient := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig())
	return NewRepository(txClient.Client())
}

func mapDBPriceToPrice(e *db.Price) (Price, error) {
	if e.Value == nil {
		return Price{}, fmt.Errorf("price %s has no value", e.ID)
	}
	return Price{
		NamespacedModel: models.NamespacedModel{
			Namespace: e.Namespace,
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: e.CreatedAt.UTC(),
			UpdatedAt: e.UpdatedAt.UTC(),
			DeletedAt: convert.SafeToUTC(e.DeletedAt),
		},
		CadencedModel: models.CadencedModel{
			ActiveFrom: e.ActiveFrom.UTC(),
			ActiveTo:   convert.SafeToUTC(e.ActiveTo),
		},
		ID:             e.ID,
		Key:            e.Key,
		SubscriptionId: e.SubscriptionID,
		PhaseKey:       e.PhaseKey,
		ItemKey:        e.ItemKey,
		Value:          *e.Value,
	}, nil
}
