package repo

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	dbsubscriptionitem "github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionitem"
	dbsubscriptionphase "github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionphase"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type subscriptionItemRepo struct {
	db *db.Client
}

var _ subscription.SubscriptionItemRepository = (*subscriptionItemRepo)(nil)

func NewSubscriptionItemRepo(db *db.Client) *subscriptionItemRepo {
	return &subscriptionItemRepo{
		db: db,
	}
}

func (r *subscriptionItemRepo) GetForSubscriptionAt(ctx context.Context, subscriptionID models.NamespacedID, at time.Time) ([]subscription.SubscriptionItem, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *subscriptionItemRepo) ([]subscription.SubscriptionItem, error) {
		items, err := repo.db.SubscriptionItem.Query().
			Where(dbsubscriptionitem.HasPhaseWith(
				dbsubscriptionphase.Or(
					dbsubscriptionphase.DeletedAtIsNil(),
					dbsubscriptionphase.DeletedAtGT(at),
				),
				dbsubscriptionphase.SubscriptionID(subscriptionID.ID),
				dbsubscriptionphase.Namespace(subscriptionID.Namespace),
			)).
			Where(dbsubscriptionitem.Or(
				dbsubscriptionitem.DeletedAtIsNil(),
				dbsubscriptionitem.DeletedAtGT(at),
			)).
			WithPhase().
			All(ctx)
		if err != nil {
			return nil, err
		}

		var result []subscription.SubscriptionItem

		for _, item := range items {
			r, err := MapDBSubscriptionItem(item)
			if err != nil {
				return nil, err
			}
			result = append(result, r)
		}

		return result, nil
	})
}

func (r *subscriptionItemRepo) GetByID(ctx context.Context, id models.NamespacedID) (subscription.SubscriptionItem, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *subscriptionItemRepo) (subscription.SubscriptionItem, error) {
		item, err := repo.db.SubscriptionItem.Query().
			Where(dbsubscriptionitem.ID(id.ID)).
			Where(dbsubscriptionitem.Namespace(id.Namespace)).
			Where(dbsubscriptionitem.Or(
				dbsubscriptionitem.DeletedAtIsNil(),
				dbsubscriptionitem.DeletedAtGT(clock.Now()),
			)).
			WithPhase().
			Only(ctx)

		if db.IsNotFound(err) {
			return subscription.SubscriptionItem{}, &subscription.ItemNotFoundError{ID: id.ID}
		}

		if err != nil {
			return subscription.SubscriptionItem{}, err
		}

		return MapDBSubscriptionItem(item)
	})
}

func (r *subscriptionItemRepo) Create(ctx context.Context, input subscription.CreateSubscriptionItemEntityInput) (subscription.SubscriptionItem, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *subscriptionItemRepo) (subscription.SubscriptionItem, error) {
		var def subscription.SubscriptionItem

		i, err := repo.db.SubscriptionItem.Create().
			SetNillableActiveFromOverrideRelativeToPhaseStart(input.ActiveFromOverrideRelativeToPhaseStart.ISOStringPtrOrNil()).
			SetNillableActiveToOverrideRelativeToPhaseStart(input.ActiveToOverrideRelativeToPhaseStart.ISOStringPtrOrNil()).
			SetActiveFrom(input.ActiveFrom).
			SetNillableActiveTo(input.ActiveTo).
			SetNamespace(input.Namespace).
			SetName(input.Name).
			SetNillableDescription(input.Description).
			SetPhaseID(input.PhaseID).
			SetKey(input.Key).
			SetName(input.RateCard.Name).
			SetNillableDescription(input.RateCard.Description).
			SetNillableFeatureKey(input.RateCard.FeatureKey).
			SetEntitlementTemplate(input.RateCard.EntitlementTemplate).
			SetTaxConfig(input.RateCard.TaxConfig).
			SetPrice(input.RateCard.Price).
			SetNillableEntitlementID(input.EntitlementID).
			SetNillableBillingCadence(input.RateCard.BillingCadence.ISOStringPtrOrNil()).
			Save(ctx)
		if err != nil {
			return def, err
		}

		return repo.GetByID(ctx, models.NamespacedID{ID: i.ID, Namespace: i.Namespace})
	})
}

func (r *subscriptionItemRepo) Delete(ctx context.Context, input models.NamespacedID) error {
	_, err := entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *subscriptionItemRepo) (any, error) {
		at := clock.Now()
		err := repo.db.SubscriptionItem.UpdateOneID(input.ID).
			Where(
				dbsubscriptionitem.Namespace(input.Namespace),
				dbsubscriptionitem.Or(
					dbsubscriptionitem.DeletedAtIsNil(),
					dbsubscriptionitem.DeletedAtGT(at),
				),
			).SetDeletedAt(at).Exec(ctx)

		if db.IsNotFound(err) {
			return nil, &subscription.ItemNotFoundError{ID: input.ID}
		}

		return nil, err
	})

	return err
}
