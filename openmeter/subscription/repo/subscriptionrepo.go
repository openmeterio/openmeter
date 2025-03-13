package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	dbplan "github.com/openmeterio/openmeter/openmeter/ent/db/plan"
	dbsubscription "github.com/openmeterio/openmeter/openmeter/ent/db/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type subscriptionRepo struct {
	db *db.Client
}

var _ subscription.SubscriptionRepository = (*subscriptionRepo)(nil)

func NewSubscriptionRepo(db *db.Client) *subscriptionRepo {
	return &subscriptionRepo{
		db: db,
	}
}

func (r *subscriptionRepo) SetEndOfCadence(ctx context.Context, id models.NamespacedID, at *time.Time) (*subscription.Subscription, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *subscriptionRepo) (*subscription.Subscription, error) {
		ent, err := repo.db.Subscription.UpdateOneID(id.ID).SetOrClearActiveTo(at).Where(dbsubscription.Namespace(id.Namespace)).Save(ctx)
		if db.IsNotFound(err) {
			return nil, &subscription.NotFoundError{
				ID: id.ID,
			}
		}
		if err != nil {
			return nil, err
		}

		sub, err := MapDBSubscription(ent)

		return lo.ToPtr(sub), err
	})
}

func (r *subscriptionRepo) GetAllForCustomerSince(ctx context.Context, customerID models.NamespacedID, at time.Time) ([]subscription.Subscription, error) {
	return entutils.TransactingRepo(
		ctx,
		r,
		func(ctx context.Context, repo *subscriptionRepo) ([]subscription.Subscription, error) {
			ents, err := repo.db.Subscription.Query().Where(
				dbsubscription.CustomerID(customerID.ID),
				dbsubscription.Namespace(customerID.Namespace),
			).Where(
				SubscriptionActiveAfter(at)...,
			).Where(
				SubscriptionNotDeletedAt(at)...,
			).All(ctx)
			if err != nil {
				return nil, err
			}

			var subs []subscription.Subscription
			for _, ent := range ents {
				sub, err := MapDBSubscription(ent)
				if err != nil {
					return nil, err
				}
				subs = append(subs, sub)
			}

			return subs, nil
		},
	)
}

func (r *subscriptionRepo) GetByID(ctx context.Context, subscriptionID models.NamespacedID) (subscription.Subscription, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *subscriptionRepo) (subscription.Subscription, error) {
		res, err := repo.db.Subscription.Query().WithPlan().Where(dbsubscription.ID(subscriptionID.ID), dbsubscription.Namespace(subscriptionID.Namespace)).Where(SubscriptionNotDeletedAt(clock.Now())...).First(ctx)

		if db.IsNotFound(err) {
			return subscription.Subscription{}, &subscription.NotFoundError{
				ID: subscriptionID.ID,
			}
		} else if err != nil {
			return subscription.Subscription{}, err
		} else if res == nil {
			return subscription.Subscription{}, fmt.Errorf("unexpected nil subscription")
		}

		return MapDBSubscription(res)
	})
}

func (r *subscriptionRepo) Create(ctx context.Context, sub subscription.CreateSubscriptionEntityInput) (subscription.Subscription, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *subscriptionRepo) (subscription.Subscription, error) {
		command := repo.db.Subscription.Create().
			SetNamespace(sub.Namespace).
			SetCustomerID(sub.CustomerId).
			SetCurrency(sub.Currency).
			SetActiveFrom(sub.ActiveFrom).
			SetName(sub.Name).
			SetNillableDescription(sub.Description).
			SetBillablesMustAlign(sub.BillablesMustAlign).
			SetMetadata(sub.Metadata)

		if sub.ActiveTo != nil {
			command = command.SetActiveTo(*sub.ActiveTo)
		}

		if sub.Plan != nil {
			command = command.SetPlanID(sub.Plan.Id)
		}

		res, err := command.Save(ctx)
		if err != nil {
			return subscription.Subscription{}, err
		}

		if res == nil {
			return subscription.Subscription{}, fmt.Errorf("unexpected nil subscription")
		}

		if res.PlanID != nil {
			plan, err := repo.db.Plan.Query().Where(dbplan.ID(*res.PlanID)).First(ctx)
			if err != nil {
				return subscription.Subscription{}, fmt.Errorf("failed to fetch plan: %w", err)
			}

			if plan == nil {
				return subscription.Subscription{}, fmt.Errorf("unexpected nil plan")
			}

			res.Edges.Plan = plan
		}

		return MapDBSubscription(res)
	})
}

func (r *subscriptionRepo) Delete(ctx context.Context, id models.NamespacedID) error {
	return entutils.TransactingRepoWithNoValue(ctx, r, func(ctx context.Context, repo *subscriptionRepo) error {
		_, err := repo.db.Subscription.UpdateOneID(id.ID).SetDeletedAt(clock.Now()).Save(ctx)
		if db.IsNotFound(err) {
			return &subscription.NotFoundError{
				ID: id.ID,
			}
		}
		if err != nil {
			return err
		}

		return nil
	})
}

func (r *subscriptionRepo) List(ctx context.Context, in subscription.ListSubscriptionsInput) (subscription.SubscriptionList, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *subscriptionRepo) (subscription.SubscriptionList, error) {
		query := repo.db.Subscription.Query().
			Where(dbsubscription.NamespaceIn(in.Namespaces...))

		if len(in.Customers) > 0 {
			query = query.Where(dbsubscription.CustomerIDIn(in.Customers...))
		}

		if in.ActiveAt != nil {
			query = query.Where(
				dbsubscription.Or(
					dbsubscription.ActiveToIsNil(),
					dbsubscription.ActiveToGT(*in.ActiveAt),
				),
			)
		}

		paged, err := query.Paginate(ctx, in.Page)
		if err != nil {
			return subscription.SubscriptionList{}, err
		}

		items, err := slicesx.MapWithErr(paged.Items, MapDBSubscription)
		if err != nil {
			return subscription.SubscriptionList{}, err
		}

		return subscription.SubscriptionList{
			Items:      items,
			Page:       paged.Page,
			TotalCount: paged.TotalCount,
		}, nil
	})
}
