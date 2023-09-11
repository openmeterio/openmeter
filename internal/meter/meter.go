package meter

import (
	"context"

	"github.com/openmeterio/openmeter/pkg/models"
)

// Repository is an interface to the meter store.
type Repository interface {
	// ListMeters returns a list of meters.
	ListMeters(ctx context.Context, namespace string) ([]models.Meter, error)

	// GetMeterByIDOrSlug returns a meter from the meter store by ID or slug.
	GetMeterByIDOrSlug(ctx context.Context, namespace string, idOrSlug string) (models.Meter, error)
}
