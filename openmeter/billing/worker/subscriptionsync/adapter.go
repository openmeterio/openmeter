package subscriptionsync

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	paginationv2 "github.com/openmeterio/openmeter/pkg/pagination/v2"
)

type Adapter interface {
	ListSubscriptionsToReconcile(context.Context, ListSubscriptionsToReconcileInput) (paginationv2.Result[SubscriptionToReconcile], error)
	ListDeletedSubscriptionsToReconcile(context.Context, ListDeletedSubscriptionsToReconcileInput) (paginationv2.Result[SubscriptionToReconcile], error)

	SyncStateAdapter

	entutils.TxCreator
}
