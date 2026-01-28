package adapter

import (
	"context"
	"fmt"
	"slices"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// ListMeters implements the [Repository] interface.
func (c *adapter) ListMeters(_ context.Context, params meter.ListMetersParams) (pagination.Result[meter.Meter], error) {
	if err := params.Validate(); err != nil {
		return pagination.Result[meter.Meter]{}, models.NewGenericValidationError(err)
	}

	meters := []meter.Meter{}

	// In memory filtering
	for _, meter := range c.getMeters() {
		if !params.WithoutNamespace && meter.Namespace != params.Namespace {
			continue
		}

		if params.SlugFilter != nil && !slices.Contains(*params.SlugFilter, meter.Key) {
			continue
		}

		meters = append(meters, meter)
	}

	// In memory pagination: case #1 if there is no pagination settings return the whole dataset
	if params.Page.IsZero() {
		return pagination.Result[meter.Meter]{
			Page:       params.Page,
			Items:      meters,
			TotalCount: len(meters),
		}, nil
	}

	// In memory pagination: case #2 if there is pagination settings return the paginated dataset
	pageNumberIndex := params.PageNumber - 1

	if pageNumberIndex*params.PageSize > len(meters) {
		meters = []meter.Meter{}
	} else {
		meters = meters[pageNumberIndex*params.PageSize:]
	}

	if len(meters) > params.PageSize {
		meters = meters[:params.PageSize]
	}

	return pagination.Result[meter.Meter]{
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

		if meter.ID == input.IDOrSlug || meter.Key == input.IDOrSlug {
			return meter, nil
		}
	}

	return meter.Meter{}, meter.NewMeterNotFoundError(input.IDOrSlug)
}

// ReplaceMeters can be used to replace all meters in the repository.
func (c *adapter) ReplaceMeters(_ context.Context, meters []meter.Meter) error {
	c.init()

	for _, m := range meters {
		if err := m.Validate(); err != nil {
			return models.NewGenericValidationError(
				fmt.Errorf("failed to validate meter: %w", err),
			)
		}
	}

	c.meters = slices.Clone(meters)

	return nil
}

// getMeters returns all meters in the memory store.
func (c *adapter) getMeters() []meter.Meter {
	c.init()

	return slices.Clone(c.meters)
}
