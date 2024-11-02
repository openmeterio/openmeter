package adapter

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/samber/lo"

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

func (r *subscriptionRepo) EndCadence(ctx context.Context, id string, at *time.Time) (*subscription.Subscription, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *subscriptionRepo) (*subscription.Subscription, error) {
		ent, err := repo.db.Subscription.UpdateOneID(id).SetNillableActiveTo(at).Save(ctx)
		if db.IsNotFound(err) {
			return nil, &subscription.NotFoundError{
				ID: id,
			}
		}
		if err != nil {
			return nil, err
		}

		sub, err := MapDBSubscription(ent)

		return lo.ToPtr(sub), err
	})
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
		res, err := repo.db.SubscriptionPatch.Query().WithValueAddItem().WithValueAddPhase().WithValueExtendPhase().WithValueRemovePhase().
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
		// Validate batchIndexes are unique.
		// This is needed as later we match the value patches to the patches via the batch index
		batchIndexes := make(map[int]struct{})
		for _, p := range patches {
			if _, exists := batchIndexes[p.BatchIndex]; exists {
				return nil, fmt.Errorf("duplicate batch index %d", p.BatchIndex)
			}
			batchIndexes[p.BatchIndex] = struct{}{}
		}

		creates, err := mapPatchesToCreates(subscriptionID, patches)
		if err != nil {
			return nil, err
		}

		// First we create the patches
		dbPatches, err := repo.db.SubscriptionPatch.MapCreateBulk(creates, func(spc *db.SubscriptionPatchCreate, i int) {
			creates[i].patch(spc)
		}).Save(ctx)
		if err != nil {
			return nil, err
		}

		// Then we create the getter util so the Value Patches can reference the patch ID
		sortedDBPatches := make([]*db.SubscriptionPatch, len(dbPatches))
		copy(sortedDBPatches, dbPatches)
		slices.SortStableFunc(sortedDBPatches, func(i, j *db.SubscriptionPatch) int {
			return i.BatchIndex - j.BatchIndex
		})

		getPatch := func(batchIndex int) *db.SubscriptionPatch {
			return sortedDBPatches[batchIndex]
		}

		// Then we create each of the values
		addItemPatch := lo.Filter(creates, func(c patchCreator, _ int) bool {
			return c.addItem != nil
		})

		if len(addItemPatch) > 0 {
			_, err = repo.db.SubscriptionPatchValueAddItem.MapCreateBulk(addItemPatch, func(spc *db.SubscriptionPatchValueAddItemCreate, i int) {
				addItemPatch[i].addItem(spc, getPatch)
			}).Save(ctx)
			if err != nil {
				return nil, err
			}
		}

		addPhasePatch := lo.Filter(creates, func(c patchCreator, _ int) bool {
			return c.addPhase != nil
		})

		if len(addPhasePatch) > 0 {
			_, err = repo.db.SubscriptionPatchValueAddPhase.MapCreateBulk(addPhasePatch, func(spc *db.SubscriptionPatchValueAddPhaseCreate, i int) {
				addPhasePatch[i].addPhase(spc, getPatch)
			}).Save(ctx)
			if err != nil {
				return nil, err
			}
		}

		removePhasePatch := lo.Filter(creates, func(c patchCreator, _ int) bool {
			return c.removePhase != nil
		})

		if len(removePhasePatch) > 0 {
			_, err = repo.db.SubscriptionPatchValueRemovePhase.MapCreateBulk(removePhasePatch, func(spc *db.SubscriptionPatchValueRemovePhaseCreate, i int) {
				removePhasePatch[i].removePhase(spc, getPatch)
			}).Save(ctx)
			if err != nil {
				return nil, err
			}
		}

		extendPhasePatch := lo.Filter(creates, func(c patchCreator, _ int) bool {
			return c.extendPhase != nil
		})

		if len(extendPhasePatch) > 0 {
			_, err = repo.db.SubscriptionPatchValueExtendPhase.MapCreateBulk(extendPhasePatch, func(spc *db.SubscriptionPatchValueExtendPhaseCreate, i int) {
				extendPhasePatch[i].extendPhase(spc, getPatch)
			}).Save(ctx)
			if err != nil {
				return nil, err
			}
		}

		// We have to refetch the patches to also load the edges, otherwise mapping would fail
		// Alternatively we could map them here in memory but that probably would not be worth the complication
		dbPatches, err = repo.db.SubscriptionPatch.Query().WithValueAddItem().WithValueAddPhase().WithValueExtendPhase().WithValueRemovePhase().Where(
			dbsubscriptionpatch.IDIn(lo.Map(dbPatches, func(p *db.SubscriptionPatch, _ int) string {
				return p.ID
			})...),
		).All(ctx)
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
