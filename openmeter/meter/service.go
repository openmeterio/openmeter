package meter

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

// Meter is an interface for the meter service.
type Service interface {
	ListMeters(ctx context.Context, params ListMetersParams) (pagination.Result[Meter], error)
	GetMeterByIDOrSlug(ctx context.Context, input GetMeterInput) (Meter, error)
}

// ManageService is an interface to manage meter service.
type ManageService interface {
	Service

	CreateMeter(ctx context.Context, input CreateMeterInput) (Meter, error)
	UpdateMeter(ctx context.Context, input UpdateMeterInput) (Meter, error)
	DeleteMeter(ctx context.Context, input DeleteMeterInput) error

	// Observer hooks
	// Useful to coordinate with other services
	RegisterPreUpdateMeterHook(hook PreUpdateMeterHook) error
}

// PreUpdateMeterHook is a hook function to be called before updating a meter.
type PreUpdateMeterHook = func(context.Context, UpdateMeterInput) error

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

	OrderBy OrderBy
	Order   sortx.Order

	Namespace string

	SlugFilter *[]string

	// WithoutNamespace is a flag to list meters without a namespace.
	// We do this instead of letting the namespace be empty to avoid
	// accidental listing of all meters across all namespaces.
	WithoutNamespace bool

	// IncludeDeleted is a flag to include deleted meters in the list.
	IncludeDeleted bool

	// Filter by event types
	EventTypes *[]string
}

// Validate validates the list meters parameters.
func (p ListMetersParams) Validate() error {
	var errs []error

	if p.Namespace == "" && !p.WithoutNamespace {
		errs = append(errs, errors.New("namespace is required"))
	}

	if p.OrderBy != "" && !slices.Contains(OrderBy("").Values(), p.OrderBy) {
		errs = append(errs, fmt.Errorf("invalid order by: %s", p.OrderBy))
	}

	if p.Order != sortx.OrderNone && (p.Order != sortx.OrderAsc && p.Order != sortx.OrderDesc) {
		errs = append(errs, fmt.Errorf("invalid order: %s", p.Order))
	}

	if p.SlugFilter != nil {
		for _, slug := range *p.SlugFilter {
			if slug == "" {
				errs = append(errs, errors.New("slug filter must not contain empty string"))
			}
		}
	}

	if p.EventTypes != nil {
		for _, eventType := range *p.EventTypes {
			if eventType == "" {
				errs = append(errs, errors.New("event type filter must not contain empty string"))
			}
		}
	}

	return errors.Join(errs...)
}

type inputOptions struct {
	AllowReservedEventTypes bool
}

var (
	_ models.Validator                         = (*CreateMeterInput)(nil)
	_ models.CustomValidator[CreateMeterInput] = (*CreateMeterInput)(nil)
)

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
	Metadata      models.Metadata
	Annotations   models.Annotations

	inputOptions
}

func (i CreateMeterInput) ValidateWith(validators ...models.ValidatorFunc[CreateMeterInput]) error {
	return models.Validate(i, validators...)
}

func ValidateCreateMeterInputWithReservedEventTypes(reserved []*EventTypePattern) models.ValidatorFunc[CreateMeterInput] {
	return func(input CreateMeterInput) error {
		if input.AllowReservedEventTypes {
			return nil
		}

		for _, pattern := range reserved {
			if pattern == nil {
				continue
			}

			if ok := pattern.MatchString(input.EventType); ok {
				return fmt.Errorf("event type '%s' is reserved: matched pattern '%s'", input.EventType, pattern.String())
			}
		}

		return nil
	}
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

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// UpdateMeterInput is a parameter object for creating a meter.
type UpdateMeterInput struct {
	ID          models.NamespacedID
	Name        string
	Description *string
	GroupBy     map[string]string
	Metadata    models.Metadata
	Annotations *models.Annotations
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
		errs = append(errs, err)
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
