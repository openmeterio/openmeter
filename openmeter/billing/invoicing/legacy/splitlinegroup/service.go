package splitlinegroup

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

type Service interface {
	DeleteSplitLineGroup(ctx context.Context, input DeleteSplitLineGroupInput) error
	UpdateSplitLineGroup(ctx context.Context, input UpdateSplitLineGroupInput) (SplitLineGroup, error)
	// GetSplitLineGroupsForSubscription returns the active split-line hierarchies required for subscription sync.
	GetSplitLineGroupsForSubscription(ctx context.Context, input billing.GetLinesForSubscriptionInput) ([]SplitLineHierarchy, error)
}
