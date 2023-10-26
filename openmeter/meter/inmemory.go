package meter

import (
	"github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/pkg/models"
)

// InMemoryRepository is an in-memory meter repository.
type InMemoryRepository = meter.InMemoryRepository

// NewInMemoryRepository returns a in-memory meter repository.
func NewInMemoryRepository(meters []models.Meter) *InMemoryRepository {
	return meter.NewInMemoryRepository(meters)
}
