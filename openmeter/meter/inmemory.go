package meter

import (
	"context"
	"slices"
	"sync"

	"github.com/openmeterio/openmeter/pkg/models"
)

// InMemoryRepository is an in-memory meter repository.
type InMemoryRepository struct {
	meters []models.Meter

	initOnce sync.Once
}

// NewInMemoryRepository returns a in-memory meter repository.
func NewInMemoryRepository(meters []models.Meter) *InMemoryRepository {
	repository := &InMemoryRepository{}
	repository.init()

	repository.meters = slices.Clone(meters)

	return repository
}

func (c *InMemoryRepository) init() {
	c.initOnce.Do(func() {
		if c.meters == nil {
			c.meters = make([]models.Meter, 0)
		}
	})
}

// ListMeters implements the [Repository] interface.
func (c *InMemoryRepository) ListAllMeters(_ context.Context) ([]models.Meter, error) {
	c.init()

	return slices.Clone(c.meters), nil
}

// ListMeters implements the [Repository] interface.
func (c *InMemoryRepository) ListMeters(_ context.Context, namespace string) ([]models.Meter, error) {
	c.init()

	return slices.Clone(c.meters), nil
}

// GetMeterByIDOrSlug implements the [Repository] interface.
func (c *InMemoryRepository) GetMeterByIDOrSlug(_ context.Context, namespace string, idOrSlug string) (models.Meter, error) {
	c.init()

	for _, meter := range c.meters {
		if meter.ID == idOrSlug || meter.Slug == idOrSlug {
			return meter, nil
		}
	}

	return models.Meter{}, &models.MeterNotFoundError{
		MeterSlug: idOrSlug,
	}
}
