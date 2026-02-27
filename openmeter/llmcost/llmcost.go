package llmcost

import (
	"errors"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/models"
)

// PriceSource identifies where a price came from.
type PriceSource string

const (
	// PriceSourceManual is used for per-namespace price overrides created by users.
	PriceSourceManual PriceSource = "manual"

	// PriceSourceSystem is used for reconciled global prices produced by the sync job.
	PriceSourceSystem PriceSource = "system"
)

func (s PriceSource) Validate() error {
	if s == "" {
		return ErrInvalidPriceSource
	}

	return nil
}

// Provider is the LLM vendor (e.g., "openai", "anthropic").
type Provider string

// ModelPricing holds the cost per token for each dimension.
type ModelPricing struct {
	// InputPerToken is the cost per input token in USD.
	InputPerToken alpacadecimal.Decimal `json:"input_per_token"`

	// OutputPerToken is the cost per output token in USD.
	OutputPerToken alpacadecimal.Decimal `json:"output_per_token"`

	// InputCachedPerToken is the cost per cached input token in USD.
	InputCachedPerToken *alpacadecimal.Decimal `json:"input_cached_per_token,omitempty"`

	// ReasoningPerToken is the cost per reasoning/thinking token in USD.
	ReasoningPerToken *alpacadecimal.Decimal `json:"reasoning_per_token,omitempty"`

	// CacheWritePerToken is the cost per cache write token in USD.
	CacheWritePerToken *alpacadecimal.Decimal `json:"cache_write_per_token,omitempty"`
}

func (p ModelPricing) Validate() error {
	var errs []error

	if p.InputPerToken.IsNegative() {
		errs = append(errs, ErrPriceMustBeNonNegative)
	}

	if p.OutputPerToken.IsNegative() {
		errs = append(errs, ErrPriceMustBeNonNegative)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// Price represents a versioned price record for an LLM model.
type Price struct {
	models.ManagedModel

	// ID is the unique identifier for this price record.
	ID string `json:"id"`

	// Namespace is nil for global prices, set for per-namespace overrides.
	Namespace *string `json:"namespace,omitempty"`

	// Provider is the LLM vendor (e.g., "openai", "anthropic").
	Provider Provider `json:"provider"`

	// ModelID is the canonical model identifier (e.g., "gpt-4", "claude-3-5-sonnet").
	ModelID string `json:"model_id"`

	// ModelName is the human-readable model name.
	ModelName string `json:"model_name"`

	// Pricing contains the cost per token for each dimension.
	Pricing ModelPricing `json:"pricing"`

	// Currency is the currency code (always "USD" for now).
	Currency string `json:"currency"`

	// Source indicates where this price came from.
	Source PriceSource `json:"source"`

	// SourcePrices stores the per-source pricing data that contributed to a system price.
	// Only populated for system prices, nil for manual overrides.
	SourcePrices SourcePricesMap `json:"source_prices,omitempty"`

	// EffectiveFrom is the time from which this price is active.
	EffectiveFrom time.Time `json:"effective_from"`

	// EffectiveTo is the time at which this price expires. Nil means current.
	EffectiveTo *time.Time `json:"effective_to,omitempty"`

	// Metadata is arbitrary key-value data.
	Metadata models.Metadata `json:"metadata,omitempty"`
}

func (p Price) Validate() error {
	var errs []error

	if p.Provider == "" {
		errs = append(errs, ErrProviderEmpty)
	}

	if p.ModelID == "" {
		errs = append(errs, ErrModelIDEmpty)
	}

	if err := p.Pricing.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := p.Source.Validate(); err != nil {
		errs = append(errs, err)
	}

	if p.EffectiveTo != nil && p.EffectiveFrom.After(*p.EffectiveTo) {
		errs = append(errs, ErrEffectiveFromAfterTo)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// SourcePrice is a price fetched from a single external source, used in-memory during sync.
type SourcePrice struct {
	Source    PriceSource  `json:"source"`
	Provider  Provider     `json:"provider"`
	ModelID   string       `json:"model_id"`
	ModelName string       `json:"model_name"`
	Pricing   ModelPricing `json:"pricing"`
	FetchedAt time.Time    `json:"fetched_at"`
}

// SourcePriceData is the per-source pricing stored as part of the JSON blob on a Price row.
type SourcePriceData struct {
	Pricing   ModelPricing `json:"pricing"`
	FetchedAt time.Time    `json:"fetched_at"`
}

// SourcePricesMap maps source name to its pricing data.
type SourcePricesMap map[PriceSource]SourcePriceData
