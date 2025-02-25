package adapter

import (
	"context"
	"slices"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// ListMeters implements the [Repository] interface.
func (c *adapter) ListMeters(_ context.Context, params meter.ListMetersParams) (pagination.PagedResponse[meter.Meter], error) {
	if err := params.Validate(); err != nil {
		return pagination.PagedResponse[meter.Meter]{}, models.NewGenericValidationError(err)
	}

	meters := []meter.Meter{}

	// In memory filtering
	for _, meter := range c.getMeters() {
		if !params.WithoutNamespace && meter.Namespace != params.Namespace {
			continue
		}

		if params.SlugFilter != nil && !slices.Contains(*params.SlugFilter, meter.Slug) {
			continue
		}

		meters = append(meters, meter)
	}

	// In memory pagination
	pageNumberIndex := params.PageNumber - 1

	if pageNumberIndex*params.PageSize > len(meters) {
		meters = []meter.Meter{}
	} else {
		meters = meters[pageNumberIndex*params.PageSize:]
	}

	if len(meters) > params.PageSize {
		meters = meters[:params.PageSize]
	}

	return pagination.PagedResponse[meter.Meter]{
		Page:       params.Page,
		Items:      meters,
		TotalCount: len(meters),
	}, nil
}

// GetMeterByIDOrSlug implements the [Repository] interface.
func (c *adapter) GetMeterByIDOrSlug(_ context.Context, input meter.GetMeterInput) (meter.Meter, error) {
	if err := input.Validate(); err != nil {
		return meter.Meter{}, models.NewGenericValidationError(err)
	}

	for _, meter := range c.getMeters() {
		if meter.Namespace != input.Namespace {
			continue
		}

		if meter.ID == input.IDOrSlug || meter.Slug == input.IDOrSlug {
			return meter, nil
		}
	}

	return meter.Meter{}, meter.NewMeterNotFoundError(input.IDOrSlug)
}

// ReplaceMeters can be used to replace all meters in the repository.
func (c *adapter) ReplaceMeters(_ context.Context, meters []meter.Meter) {
	c.init()

	c.meters = slices.Clone(meters)
}

// getMeters returns all meters in the memory store.
func (c *adapter) getMeters() []meter.Meter {
	c.init()

	return slices.Clone(c.meters)
}
