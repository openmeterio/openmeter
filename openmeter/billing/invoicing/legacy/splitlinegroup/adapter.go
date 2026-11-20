package splitlinegroup

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Adapter interface {
	entutils.TxCreator
	GetSplitLineGroupsForSubscription(ctx context.Context, input GetSplitLineGroupsForSubscriptionInput) ([]SplitLineHierarchy, error)
	CreateSplitLineGroup(ctx context.Context, input SplitLineGroupCreate) (SplitLineGroup, error)
	UpdateSplitLineGroup(ctx context.Context, input SplitLineGroupUpdate) (SplitLineGroup, error)
	DeleteSplitLineGroup(ctx context.Context, input models.NamespacedID) error
	GetSplitLineGroup(ctx context.Context, input models.NamespacedID) (SplitLineHierarchy, error)
	GetSplitLineGroups(ctx context.Context, input GetSplitLineGroupsInput) ([]SplitLineHierarchy, error)
}
