package splitlinegroup

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/models"
)

type Service interface {
	CreateSplitLineGroup(ctx context.Context, input SplitLineGroupCreate) (SplitLineGroup, error)
	GetSplitLineGroups(ctx context.Context, input GetSplitLineGroupsInput) ([]SplitLineHierarchy, error)
	DeleteSplitLineGroup(ctx context.Context, input models.NamespacedID) error
	UpdateSplitLineGroup(ctx context.Context, input SplitLineGroupUpdate) (SplitLineGroup, error)
	// GetSplitLineGroupsForSubscription returns the active split-line hierarchies required for subscription sync.
	GetSplitLineGroupsForSubscription(ctx context.Context, input GetSplitLineGroupsForSubscriptionInput) ([]SplitLineHierarchy, error)
	GetSplitLineGroup(ctx context.Context, input models.NamespacedID) (SplitLineHierarchy, error)
}
