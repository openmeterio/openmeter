package repo

import (
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	dbsubscriptionitem "github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionitem"
	dbsubscriptionphase "github.com/openmeterio/openmeter/openmeter/ent/db/subscriptionphase"
	"github.com/openmeterio/openmeter/openmeter/subscription"
)

type subscriptionReadScope struct {
	namespace string
	at        time.Time
}

type subscriptionReadScopes map[subscriptionReadScope][]string

// Subscriptions expanded together share a namespace and cutoff. Group their IDs so
// the item query can look up all matching phases with a single EXISTS.
func groupSubscriptionReads(inputs []subscription.GetForSubscriptionAtInput) subscriptionReadScopes {
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

func (scope subscriptionReadScope) itemFilter(subscriptionIDs []string) predicate.SubscriptionItem {
	return dbsubscriptionitem.And(
		// Scope items explicitly so PostgreSQL can use the (namespace, phase_id, key) index.
		dbsubscriptionitem.Namespace(scope.namespace),
		dbsubscriptionitem.HasPhaseWith(scope.phaseFilter(subscriptionIDs)),
		dbsubscriptionitem.Or(
			dbsubscriptionitem.DeletedAtIsNil(),
			dbsubscriptionitem.DeletedAtGT(scope.at),
		),
	)
}

func (scopes subscriptionReadScopes) phaseFilter() predicate.SubscriptionPhase {
	return dbsubscriptionphase.Or(lo.MapToSlice(scopes, subscriptionReadScope.phaseFilter)...)
}

func (scopes subscriptionReadScopes) itemFilter() predicate.SubscriptionItem {
	return dbsubscriptionitem.Or(lo.MapToSlice(scopes, subscriptionReadScope.itemFilter)...)
}
