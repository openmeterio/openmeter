package meter

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/pkg/pagination"
)

// Meter is an interface to the meter store.
type Service interface {
	// ListMeters returns a list of meters for the given namespace.
	ListMeters(ctx context.Context, params ListMetersParams) (pagination.PagedResponse[Meter], error)

	// GetMeterByIDOrSlug returns a meter from the meter store by ID or slug.
	GetMeterByIDOrSlug(ctx context.Context, namespace string, idOrSlug string) (Meter, error)
}

// ListMetersParams is a parameter object for listing meters.
type ListMetersParams struct {
	pagination.Page
	Namespace string

	// WithoutNamespace is a flag to list meters without a namespace.
	// We do this instead of letting the namespace be empty to avoid
	// accidental listing of all meters across all namespaces.
	WithoutNamespace bool
}

// Validate validates the list meters parameters.
func (p ListMetersParams) Validate() error {
	var errs []error

	if p.Namespace == "" && !p.WithoutNamespace {
		errs = append(errs, errors.New("namespace is required"))
	}

	if err := p.Page.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid pagination: %w", err))
	}

	return errors.Join(errs...)
}
