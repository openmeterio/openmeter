package price

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/models"
)

type Connector interface {
	Create(ctx context.Context, input CreateInput) (*Price, error)
	GetForSubscription(ctx context.Context, subscriptionID models.NamespacedID) ([]Price, error)
}

type connector struct {
	repo Repository
}

func NewConnector(repo Repository) Connector {
	return &connector{repo}
}

func (c *connector) Create(ctx context.Context, input CreateInput) (*Price, error) {
	return c.repo.Create(ctx, input)
}

func (c *connector) GetForSubscription(ctx context.Context, subscriptionID models.NamespacedID) ([]Price, error) {
	// Should we return deleted prices?
	return c.repo.GetForSubscription(ctx, subscriptionID, GetPriceFilters{IncludeDeleted: false})
}
