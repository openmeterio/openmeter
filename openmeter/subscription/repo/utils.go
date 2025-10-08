package repo

import (
	"time"

	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	db_subscription "github.com/openmeterio/openmeter/openmeter/ent/db/subscription"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func SubscriptionActiveAfter(at time.Time) []predicate.Subscription {
	return []predicate.Subscription{
		db_subscription.Or(
			db_subscription.ActiveToIsNil(),
			db_subscription.ActiveToGT(at),
		),
	}
}

func SubscriptionActiveAt(at time.Time) []predicate.Subscription {
	return []predicate.Subscription{
		db_subscription.ActiveFromLTE(at),
		db_subscription.Or(
			db_subscription.ActiveToIsNil(),
			db_subscription.ActiveToGT(at),
		),
	}
}

// If subscription is active at any point in the period then it's active
func SubscriptionActiveInPeriod(period timeutil.StartBoundedPeriod) []predicate.Subscription {
	predicates := SubscriptionActiveAfter(period.From)

	if period.To != nil {
		predicates = append(predicates, db_subscription.ActiveFromLT(*period.To))
	}

	return predicates
}

func SubscriptionNotDeletedAt(at time.Time) []predicate.Subscription {
	return []predicate.Subscription{
		db_subscription.Or(
			db_subscription.DeletedAtGT(at),
			db_subscription.DeletedAtIsNil(),
		),
	}
}
