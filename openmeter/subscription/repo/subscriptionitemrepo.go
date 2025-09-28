package repo

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	dbsubscriptionitem "github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionitem"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
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

func getItemForSubscriptionAtFilter(input subscription.GetForSubscriptionAtInput) predicate.SubscriptionItem {
	return dbsubscriptionitem.And(
		dbsubscriptionitem.HasPhaseWith(getPhaseForSubscriptionAtFilter(input)),
		dbsubscriptionitem.Or(
			dbsubscriptionitem.DeletedAtIsNil(),
			dbsubscriptionitem.DeletedAtGT(input.At),
		),
	)
}

func (r *subscriptionItemRepo) GetForSubscriptionAt(ctx context.Context, input subscription.GetForSubscriptionAtInput) ([]subscription.SubscriptionItem, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *subscriptionItemRepo) ([]subscription.SubscriptionItem, error) {
		items, err := repo.db.SubscriptionItem.Query().
			Where(getItemForSubscriptionAtFilter(input)).
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

func (r *subscriptionItemRepo) GetForSubscriptionsAt(ctx context.Context, input []subscription.GetForSubscriptionAtInput) ([]subscription.SubscriptionItem, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *subscriptionItemRepo) ([]subscription.SubscriptionItem, error) {
		if len(input) == 0 {
			return nil, fmt.Errorf("filter is empty")
		}

		items, err := repo.db.SubscriptionItem.Query().
			Where(dbsubscriptionitem.Or(
				slicesx.Map(input, getItemForSubscriptionAtFilter)...,
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
			return subscription.SubscriptionItem{}, subscription.NewItemNotFoundError(id.ID)
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

		cmd := repo.db.SubscriptionItem.Create().
			SetNillableActiveFromOverrideRelativeToPhaseStart(input.ActiveFromOverrideRelativeToPhaseStart.ISOStringPtrOrNil()).
			SetNillableActiveToOverrideRelativeToPhaseStart(input.ActiveToOverrideRelativeToPhaseStart.ISOStringPtrOrNil()).
			SetActiveFrom(input.ActiveFrom).
			SetNillableActiveTo(input.ActiveTo).
			SetNamespace(input.Namespace).
			SetName(input.Name).
			SetNillableDescription(input.Description).
			SetPhaseID(input.PhaseID).
			SetKey(input.Key).
			SetName(input.RateCard.AsMeta().Name).
			SetNillableDescription(input.RateCard.AsMeta().Description).
			SetNillableFeatureKey(input.RateCard.AsMeta().FeatureKey).
			SetNillableEntitlementID(input.EntitlementID).
			SetNillableBillingCadence(input.RateCard.GetBillingCadence().ISOStringPtrOrNil()).
			SetNillableRestartsBillingPeriod(input.BillingBehaviorOverride.RestartBillingPeriod)

		if input.Annotations != nil {
			cmd.SetAnnotations(input.Annotations)
		}

		// Due to the custom value scanner, these fields don't have Nillable setters generated, and the normal setters panic when trying to call .Validate() on nil
		if input.RateCard.AsMeta().EntitlementTemplate != nil {
			cmd.SetEntitlementTemplate(input.RateCard.AsMeta().EntitlementTemplate)
		}

		if input.RateCard.AsMeta().TaxConfig != nil {
			cmd.SetTaxConfig(input.RateCard.AsMeta().TaxConfig)
		}

		if input.RateCard.AsMeta().Price != nil {
			cmd.SetPrice(input.RateCard.AsMeta().Price)
		}

		if !input.RateCard.AsMeta().Discounts.IsEmpty() {
			cmd.SetDiscounts(lo.EmptyableToPtr(input.RateCard.AsMeta().Discounts))
		}

		i, err := cmd.Save(ctx)
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
			return nil, subscription.NewItemNotFoundError(input.ID)
		}

		return nil, err
	})

	return err
}
