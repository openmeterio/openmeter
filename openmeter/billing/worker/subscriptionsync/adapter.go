package subscriptionsync

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	paginationv2 "github.com/openmeterio/openmeter/pkg/pagination/v2"
)

type Adapter interface {
	ListSubscriptionsToReconcile(context.Context, ListSubscriptionsToReconcileInput) (paginationv2.Result[models.NamespacedID], error)
	ListDeletedSubscriptionsToReconcile(context.Context, ListDeletedSubscriptionsToReconcileInput) (paginationv2.Result[models.NamespacedID], error)

	SyncStateV2Adapter

	entutils.TxCreator
}
