package adapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	dbsubscription "github.com/openmeterio/openmeter/openmeter/ent/db/subscription"
	dbsubscriptionpatch "github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionpatch"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type subscriptionRepo struct {
	db *db.Client
}

var _ subscription.Repository = (*subscriptionRepo)(nil)

func NewSubscriptionRepo(db *db.Client) *subscriptionRepo {
	return &subscriptionRepo{
		db: db,
	}
}

func (r *subscriptionRepo) GetCustomerSubscription(ctx context.Context, customerID models.NamespacedID) (subscription.Subscription, error) {
	return entutils.TransactingRepo(
		ctx,
		r,
		func(ctx context.Context, repo *subscriptionRepo) (subscription.Subscription, error) {
			now := clock.Now()
			res, err := repo.db.Subscription.Query().Where(
				dbsubscription.CustomerID(customerID.ID),
				dbsubscription.Namespace(customerID.Namespace),
			).Where(
				subscriptionActiveAt(now)...,
			).Where(
				subscriptionNotDeletedAt(now)...,
			).First(ctx)

			if db.IsNotFound(err) {
				return subscription.Subscription{}, &subscription.NotFoundError{
					CustomerID: customerID.ID,
				}
			} else if err != nil {
				return subscription.Subscription{}, err
			} else if res == nil {
				return subscription.Subscription{}, fmt.Errorf("unexpected nil subscription")
			}

			return MapDBSubscription(res)
		},
	)
}

func (r *subscriptionRepo) GetSubscription(ctx context.Context, subscriptionID models.NamespacedID) (subscription.Subscription, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *subscriptionRepo) (subscription.Subscription, error) {
		res, err := repo.db.Subscription.Query().Where(dbsubscription.ID(subscriptionID.ID), dbsubscription.Namespace(subscriptionID.Namespace)).First(ctx)

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

func (r *subscriptionRepo) CreateSubscription(ctx context.Context, namespace string, sub subscription.CreateSubscriptionInput) (subscription.Subscription, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *subscriptionRepo) (subscription.Subscription, error) {
		command := repo.db.Subscription.Create().
			SetNamespace(namespace).
			SetPlanKey(sub.Plan.Key).
			SetPlanVersion(sub.Plan.Version).
			SetCustomerID(sub.CustomerId).
			SetCurrency(sub.Currency).
			SetActiveFrom(sub.ActiveFrom)

		if sub.ActiveTo != nil {
			command = command.SetActiveTo(*sub.ActiveTo)
		}

		res, err := command.Save(ctx)
		if err != nil {
			return subscription.Subscription{}, err
		}

		if res == nil {
			return subscription.Subscription{}, fmt.Errorf("unexpected nil subscription")
		}

		return MapDBSubscription(res)
	})
}

func (r *subscriptionRepo) GetSubscriptionPatches(ctx context.Context, subscriptionID models.NamespacedID) ([]subscription.SubscriptionPatch, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *subscriptionRepo) ([]subscription.SubscriptionPatch, error) {
		// Should we validate that the subscription is active?
		res, err := repo.db.SubscriptionPatch.Query().WithValueAddItem().WithValueAddPhase().
			Where(
				dbsubscriptionpatch.SubscriptionID(subscriptionID.ID),
			).All(ctx)
		if err != nil {
			return nil, err
		}

		patches := make([]subscription.SubscriptionPatch, 0, len(res))
		for _, p := range res {
			sp, err := MapDBSubscriptionPatch(p)
			if err != nil {
				return nil, err
			}
			patches = append(patches, sp)
		}

		return patches, nil
	})
}

func (r *subscriptionRepo) CreateSubscriptionPatches(ctx context.Context, subscriptionID models.NamespacedID, patches []subscription.CreateSubscriptionPatchInput) ([]subscription.SubscriptionPatch, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *subscriptionRepo) ([]subscription.SubscriptionPatch, error) {
		dbPatches, err := repo.db.SubscriptionPatch.MapCreateBulk(patches, func(spc *db.SubscriptionPatchCreate, i int) {
			spc.
				SetSubscriptionID(subscriptionID.ID).
				SetNamespace(subscriptionID.Namespace).
				SetAppliedAt(patches[i].AppliedAt).
				SetBatchIndex(patches[i].BatchIndex).
				SetOperation(string(patches[i].Op())).
				SetPath(string(patches[i].Path()))

			if patches[i].Op() == subscription.PatchOperationAdd && patches[i].Path().Type() == subscription.PatchPathTypeItem {
				spc.SetValueAddItem(&db.SubscriptionPatchValueAddItem{
					// TODO: map
				})
				panic("mapping not implemented")
			} else if patches[i].Op() == subscription.PatchOperationAdd && patches[i].Path().Type() == subscription.PatchPathTypePhase {
				spc.SetValueAddPhase(&db.SubscriptionPatchValueAddPhase{
					// TODO: map
				})
				panic("mapping not implemented")
			} else if patches[i].Op() == subscription.PatchOperationExtend && patches[i].Path().Type() == subscription.PatchPathTypePhase {
				spc.SetValueExtendPhase(&db.SubscriptionPatchValueExtendPhase{
					// TODO: map
				})
				panic("mapping not implemented")
			}
		}).Save(ctx)
		if err != nil {
			return nil, err
		}

		patches := make([]subscription.SubscriptionPatch, 0, len(dbPatches))
		for _, p := range dbPatches {
			sp, err := MapDBSubscriptionPatch(p)
			if err != nil {
				return nil, err
			}
			patches = append(patches, sp)
		}

		return patches, nil
	})
}
