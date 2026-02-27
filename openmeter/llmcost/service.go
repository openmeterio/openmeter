package llmcost

import (
	"context"
	"errors"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// Service provides read-only access to LLM cost prices and management of per-namespace overrides.
type Service interface {
	// ListPrices returns global (synced) prices with optional filtering.
	ListPrices(ctx context.Context, input ListPricesInput) (pagination.Result[Price], error)

	// GetPrice returns a specific price by ID.
	GetPrice(ctx context.Context, input GetPriceInput) (Price, error)

	// ResolvePrice returns the effective price for a model in a namespace,
	// preferring namespace overrides over global prices.
	ResolvePrice(ctx context.Context, input ResolvePriceInput) (Price, error)

	// CreateOverride creates a per-namespace price override.
	CreateOverride(ctx context.Context, input CreateOverrideInput) (Price, error)

	// UpdateOverride updates an existing per-namespace price override.
	UpdateOverride(ctx context.Context, input UpdateOverrideInput) (Price, error)

	// DeleteOverride soft-deletes a per-namespace price override.
	DeleteOverride(ctx context.Context, input DeleteOverrideInput) error

	// ListOverrides returns per-namespace price overrides.
	ListOverrides(ctx context.Context, input ListOverridesInput) (pagination.Result[Price], error)
}

var (
	_ models.Validator = (*ListPricesInput)(nil)
	_ models.Validator = (*GetPriceInput)(nil)
	_ models.Validator = (*ResolvePriceInput)(nil)
	_ models.Validator = (*CreateOverrideInput)(nil)
	_ models.Validator = (*UpdateOverrideInput)(nil)
	_ models.Validator = (*DeleteOverrideInput)(nil)
	_ models.Validator = (*ListOverridesInput)(nil)
)

// ListPricesInput filters for listing global prices.
type ListPricesInput struct {
	pagination.Page

	// Namespace is used to overlay namespace overrides on top of global prices.
	// When set, any global price that has a matching override in this namespace
	// will be replaced by the override in the result.
	Namespace string `json:"namespace,omitempty"`

	// Provider filters by LLM vendor.
	Provider *Provider `json:"provider,omitempty"`

	// ModelID filters by canonical model identifier.
	ModelID *string `json:"model_id,omitempty"`

	// At queries prices effective at a specific point in time.
	At *time.Time `json:"at,omitempty"`
}

func (i ListPricesInput) Validate() error {
	var errs []error

	if !i.Page.IsZero() {
		if err := i.Page.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// GetPriceInput identifies a price by ID.
type GetPriceInput struct {
	ID string
}

func (i GetPriceInput) Validate() error {
	if i.ID == "" {
		return ErrPriceIDEmpty
	}

	return nil
}

// ResolvePriceInput resolves the effective price for a model in a namespace.
type ResolvePriceInput struct {
	Namespace string
	Provider  Provider
	ModelID   string
	At        *time.Time // defaults to now if nil
}

func (i ResolvePriceInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, ErrNamespaceEmpty)
	}

	if i.Provider == "" {
		errs = append(errs, ErrProviderEmpty)
	}

	if i.ModelID == "" {
		errs = append(errs, ErrModelIDEmpty)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// CreateOverrideInput creates a per-namespace price override.
type CreateOverrideInput struct {
	Namespace     string
	Provider      Provider
	ModelID       string
	ModelName     string
	Pricing       ModelPricing
	EffectiveFrom time.Time
	EffectiveTo   *time.Time
}

func (i CreateOverrideInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, ErrNamespaceEmpty)
	}

	if i.Provider == "" {
		errs = append(errs, ErrProviderEmpty)
	}

	if i.ModelID == "" {
		errs = append(errs, ErrModelIDEmpty)
	}

	if err := i.Pricing.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.EffectiveTo != nil && i.EffectiveFrom.After(*i.EffectiveTo) {
		errs = append(errs, ErrEffectiveFromAfterTo)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// UpdateOverrideInput updates an existing per-namespace price override.
type UpdateOverrideInput struct {
	ID          string
	Namespace   string
	Pricing     ModelPricing
	EffectiveTo *time.Time
}

func (i UpdateOverrideInput) Validate() error {
	var errs []error

	if i.ID == "" {
		errs = append(errs, ErrPriceIDEmpty)
	}

	if i.Namespace == "" {
		errs = append(errs, ErrNamespaceEmpty)
	}

	if err := i.Pricing.Validate(); err != nil {
		errs = append(errs, err)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// DeleteOverrideInput identifies a per-namespace override to delete.
type DeleteOverrideInput struct {
	ID        string
	Namespace string
}

func (i DeleteOverrideInput) Validate() error {
	var errs []error

	if i.ID == "" {
		errs = append(errs, ErrPriceIDEmpty)
	}

	if i.Namespace == "" {
		errs = append(errs, ErrNamespaceEmpty)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// ListOverridesInput filters for listing per-namespace overrides.
type ListOverridesInput struct {
	Namespace string
	pagination.Page

	Provider *Provider `json:"provider,omitempty"`
	ModelID  *string   `json:"model_id,omitempty"`
}

func (i ListOverridesInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, ErrNamespaceEmpty)
	}

	if !i.Page.IsZero() {
		if err := i.Page.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
