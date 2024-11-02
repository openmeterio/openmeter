package price

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

type Connector interface {
	Create(ctx context.Context, input CreateInput) (*Price, error)
	Delete(ctx context.Context, id models.NamespacedID) error
	GetForSubscription(ctx context.Context, subscriptionID models.NamespacedID) ([]Price, error)
	EndCadence(ctx context.Context, id models.NamespacedID, at *time.Time) (*Price, error)
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

func (c *connector) EndCadence(ctx context.Context, id models.NamespacedID, at *time.Time) (*Price, error) {
	return c.repo.EndCadence(ctx, id.ID, at)
}

func (c *connector) GetForSubscription(ctx context.Context, subscriptionID models.NamespacedID) ([]Price, error) {
	// Should we return deleted prices?
	return c.repo.GetForSubscription(ctx, subscriptionID, GetPriceFilters{IncludeDeleted: false})
}

func (c *connector) Delete(ctx context.Context, id models.NamespacedID) error {
	return c.repo.Delete(ctx, id)
}
