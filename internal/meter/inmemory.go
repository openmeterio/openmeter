package meter

import (
	"context"
	"sync"

	"github.com/openmeterio/openmeter/pkg/models"
)

// InMemoryRepository is an in-memory meter repository.
type InMemoryRepository struct {
	meters map[string]models.Meter

	mu       sync.Mutex
	initOnce sync.Once
}

// NewInMemoryRepository returns a in-memory meter repository.
func NewInMemoryRepository(meters []models.Meter) *InMemoryRepository {
	repository := &InMemoryRepository{}
	repository.init()

	for _, meter := range meters {
		repository.meters[meter.Slug] = meter
	}

	return repository
}

func (c *InMemoryRepository) init() {
	c.initOnce.Do(func() {
		c.meters = make(map[string]models.Meter)
	})
}

// GetMeterBySlug implements the [Repository] interface.
func (c *InMemoryRepository) GetMeterBySlug(_ context.Context, namespace string, slug string) (models.Meter, error) {
	c.init()

	c.mu.Lock()
	defer c.mu.Unlock()

	meter, ok := c.meters[slug]
	if !ok {
		return models.Meter{}, &models.MeterNotFoundError{
			MeterSlug: slug,
		}
	}

	return meter, nil
}
