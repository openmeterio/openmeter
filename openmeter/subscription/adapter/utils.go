package adapter

import (
	"time"

	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	db_subscription "github.com/openmeterio/openmeter/openmeter/ent/db/subscription"
)

func subscriptionActiveAt(at time.Time) []predicate.Subscription {
	return []predicate.Subscription{
		db_subscription.Or(
			db_subscription.ActiveFromLTE(at),
		),
		db_subscription.Or(
			db_subscription.ActiveToIsNil(),
			db_subscription.ActiveToGT(at),
		),
	}
}

func subscriptionNotDeletedAt(at time.Time) []predicate.Subscription {
	return []predicate.Subscription{
		db_subscription.Or(
			db_subscription.DeletedAtGT(at),
			db_subscription.DeletedAtIsNil(),
		),
	}
}
