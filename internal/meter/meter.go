package meter

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/models"
)

// Repository is an interface to the meter store.
type Repository interface {
	// GetMeterBySlug returns a meter from the meter store by slug.
	GetMeterBySlug(ctx context.Context, namespace string, slug string) (models.Meter, error)
}
