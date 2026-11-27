package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	dbsubscriptionphase "github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionphase"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type subscriptionPhaseRepo struct {
	db *db.Client
}

var _ subscription.SubscriptionPhaseRepository = (*subscriptionPhaseRepo)(nil)

func NewSubscriptionPhaseRepo(db *db.Client) *subscriptionPhaseRepo {
	return &subscriptionPhaseRepo{
		db: db,
	}
}

type subscriptionReadScope struct {
	namespace string
	at        time.Time
}

// Subscriptions expanded together share a namespace and cutoff. Group their IDs so
// the item query can look up all matching phases with a single EXISTS.
func groupSubscriptionReads(inputs []subscription.GetForSubscriptionAtInput) map[subscriptionReadScope][]string {
	return lo.GroupByMap(inputs, func(input subscription.GetForSubscriptionAtInput) (subscriptionReadScope, string) {
		return subscriptionReadScope{namespace: input.Namespace, at: input.At.UTC()}, input.SubscriptionID
	})
}

func (scope subscriptionReadScope) phaseFilter(subscriptionIDs []string) predicate.SubscriptionPhase {
	return dbsubscriptionphase.And(
		dbsubscriptionphase.SubscriptionIDIn(subscriptionIDs...),
		dbsubscriptionphase.Namespace(scope.namespace),
		dbsubscriptionphase.Or(
			dbsubscriptionphase.DeletedAtIsNil(),
			dbsubscriptionphase.DeletedAtGT(scope.at),
		),
	)
}

func (r *subscriptionPhaseRepo) GetForSubscriptionAt(ctx context.Context, input subscription.GetForSubscriptionAtInput) ([]subscription.SubscriptionPhase, error) {
	return r.GetForSubscriptionsAt(ctx, []subscription.GetForSubscriptionAtInput{input})
}

func (r *subscriptionPhaseRepo) GetForSubscriptionsAt(ctx context.Context, input []subscription.GetForSubscriptionAtInput) ([]subscription.SubscriptionPhase, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *subscriptionPhaseRepo) ([]subscription.SubscriptionPhase, error) {
		if len(input) == 0 {
			return nil, fmt.Errorf("filter is empty")
		}

		phases, err := repo.db.SubscriptionPhase.Query().
			Where(
				dbsubscriptionphase.Or(
					lo.MapToSlice(groupSubscriptionReads(input), subscriptionReadScope.phaseFilter)...,
				),
			).
			All(ctx)
		if err != nil {
			return nil, err
		}

		var result []subscription.SubscriptionPhase

		for _, phase := range phases {
			r, err := MapDBSubscripitonPhase(phase)
			if err != nil {
				return nil, err
			}

			result = append(result, r)
		}

		return result, nil
	})
}

func (r *subscriptionPhaseRepo) Create(ctx context.Context, phase subscription.CreateSubscriptionPhaseEntityInput) (subscription.SubscriptionPhase, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *subscriptionPhaseRepo) (subscription.SubscriptionPhase, error) {
		var def subscription.SubscriptionPhase

		dbPhase, err := repo.db.SubscriptionPhase.Create().
			SetActiveFrom(phase.ActiveFrom).
			SetNillableDescription(phase.Description).
			SetKey(phase.Key).
			SetName(phase.Name).
			SetMetadata(phase.Metadata).
			SetNamespace(phase.Namespace).
			SetSubscriptionID(phase.SubscriptionID).
			SetNillableSortHint(phase.SortHint).
			Save(ctx)
		if err != nil {
			return def, err
		}

		return MapDBSubscripitonPhase(dbPhase)
	})
}

func (r *subscriptionPhaseRepo) Delete(ctx context.Context, id models.NamespacedID) error {
	_, err := entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *subscriptionPhaseRepo) (any, error) {
		at := clock.Now()
		err := repo.db.SubscriptionPhase.UpdateOneID(id.ID).
			Where(
				dbsubscriptionphase.Namespace(id.Namespace),
				dbsubscriptionphase.Or(
					dbsubscriptionphase.DeletedAtIsNil(),
					dbsubscriptionphase.DeletedAtGT(at),
				),
			).SetDeletedAt(at).Exec(ctx)
		if db.IsNotFound(err) {
			return nil, subscription.NewPhaseNotFoundError(id.ID)
		}

		return nil, err
	})

	return err
}
