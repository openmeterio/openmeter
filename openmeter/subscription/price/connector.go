package price

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/models"
)

type Connector interface {
	Create(ctx context.Context, input CreateInput) (*Price, error)
	GetForSubscription(ctx context.Context, subscriptionID models.NamespacedID) ([]Price, error)
}
