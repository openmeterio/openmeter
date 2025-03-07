package meter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// Meter is an interface for the meter service.
type Service interface {
	ListMeters(ctx context.Context, params ListMetersParams) (pagination.PagedResponse[Meter], error)
	GetMeterByIDOrSlug(ctx context.Context, input GetMeterInput) (Meter, error)
}

// ManageService is an interface to manage meter service.
type ManageService interface {
	Service

	CreateMeter(ctx context.Context, input CreateMeterInput) (Meter, error)
	UpdateMeter(ctx context.Context, input UpdateMeterInput) (Meter, error)
	DeleteMeter(ctx context.Context, input DeleteMeterInput) error
}

// GetMeterInput is a parameter object for getting a meter.
type GetMeterInput struct {
	Namespace string
	IDOrSlug  string
}

// Validate validates the get meter input.
func (i GetMeterInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.IDOrSlug == "" {
		errs = append(errs, errors.New("id or slug is required"))
	}

	return errors.Join(errs...)
}

// ListMetersParams is a parameter object for listing meters.
type ListMetersParams struct {
	pagination.Page
	Namespace string

	SlugFilter *[]string

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

	if p.SlugFilter != nil {
		for _, slug := range *p.SlugFilter {
			if slug == "" {
				errs = append(errs, errors.New("slug filter must not contain empty string"))
				break
			}
		}
	}

	return errors.Join(errs...)
}

// CreateMeterInput is a parameter object for creating a meter.
type CreateMeterInput struct {
	Namespace     string
	Name          string
	Key           string
	Description   *string
	Aggregation   MeterAggregation
	EventType     string
	EventFrom     *time.Time
	ValueProperty *string
	GroupBy       map[string]string
}

// Validate validates the create meter input.
func (i CreateMeterInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.Name == "" {
		errs = append(errs, errors.New("name is required"))
	}

	if i.Key == "" {
		errs = append(errs, errors.New("key is required"))
	}

	if i.Description != nil && *i.Description == "" {
		errs = append(errs, errors.New("description must not be empty string"))
	}

	if i.Aggregation == "" {
		errs = append(errs, errors.New("meter aggregation is required"))
	}

	if i.EventType == "" {
		errs = append(errs, errors.New("meter event type is required"))
	}

	if i.EventFrom != nil && i.EventFrom.IsZero() {
		errs = append(errs, errors.New("meter event from must not be zero"))
	}

	// Validate aggregation
	if err := validateMeterAggregation(i.ValueProperty, i.Aggregation); err != nil {
		errs = append(errs, fmt.Errorf("invalid meter aggregation: %w", err))
	}

	// Validate group by values
	if err := validateMeterGroupBy(i.ValueProperty, i.GroupBy); err != nil {
		errs = append(errs, fmt.Errorf("invalid meter group by: %w", err))
	}

	return errors.Join(errs...)
}

// UpdateMeterInput is a parameter object for creating a meter.
type UpdateMeterInput struct {
	ID          models.NamespacedID
	Name        string
	Description *string
	GroupBy     map[string]string
}

// Validate validates the create meter input.
func (i UpdateMeterInput) Validate(valueProperty *string) error {
	var errs []error

	if err := i.ID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invalid meter id: %w", err))
	}

	if i.Name == "" {
		errs = append(errs, errors.New("name is required"))
	}

	if i.Description != nil && *i.Description == "" {
		errs = append(errs, errors.New("description must not be empty string"))
	}

	err := validateMeterGroupBy(valueProperty, i.GroupBy)
	if err != nil {
		return err
	}

	return errors.Join(errs...)
}

// DeleteMeterInput is a parameter object for deleting a meter.
type DeleteMeterInput struct {
	Namespace string
	IDOrSlug  string
}

// Validate validates the delete meter input.
func (i DeleteMeterInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.IDOrSlug == "" {
		errs = append(errs, errors.New("id or slug is required"))
	}

	return errors.Join(errs...)
}
