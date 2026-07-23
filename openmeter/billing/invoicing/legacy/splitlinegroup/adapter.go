package splitlinegroup

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Adapter interface {
	GetSplitLineGroupsForSubscription(ctx context.Context, input billing.GetLinesForSubscriptionInput) ([]SplitLineHierarchy, error)
	CreateSplitLineGroup(ctx context.Context, input CreateSplitLineGroupAdapterInput) (SplitLineGroup, error)
	UpdateSplitLineGroup(ctx context.Context, input UpdateSplitLineGroupInput) (SplitLineGroup, error)
	DeleteSplitLineGroup(ctx context.Context, input DeleteSplitLineGroupInput) error
	GetSplitLineGroup(ctx context.Context, input GetSplitLineGroupInput) (SplitLineHierarchy, error)
	GetSplitLineGroupHeaders(ctx context.Context, input GetSplitLineGroupHeadersInput) (SplitLineGroupHeaders, error)

	entutils.TxCreator
}
