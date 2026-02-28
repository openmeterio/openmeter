package feature

import (
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/llmcost"
	"github.com/openmeterio/openmeter/openmeter/meter"
)

// UnitCostType identifies the type of unit cost.
type UnitCostType string

const (
	UnitCostTypeLLM    UnitCostType = "llm"
	UnitCostTypeManual UnitCostType = "manual"
)

// UnitCost represents an optional per-unit cost configuration for a feature.
type UnitCost struct {
	// Type is the unit cost type: "llm" or "manual".
	Type UnitCostType `json:"type"`

	// Manual is set when type is "manual".
	Manual *ManualUnitCost `json:"manual,omitempty"`

	// LLM is set when type is "llm".
	LLM *LLMUnitCost `json:"llm,omitempty"`
}

// ManualUnitCost is a fixed per-unit cost amount.
type ManualUnitCost struct {
	// Amount is the per-unit cost in USD.
	Amount alpacadecimal.Decimal `json:"amount"`
}

// LLMUnitCost configures dynamic cost lookup from the LLM cost database.
// For each dimension (provider, model, token type), either a static value
// or a meter group-by property name can be specified (mutually exclusive).
type LLMUnitCost struct {
	// ProviderProperty is the meter group-by key that holds the LLM provider value.
	// Mutually exclusive with Provider.
	ProviderProperty string `json:"provider_property,omitempty"`

	// Provider is a static LLM provider value (e.g. "openai", "anthropic").
	// Mutually exclusive with ProviderProperty.
	Provider string `json:"provider,omitempty"`

	// ModelProperty is the meter group-by key that holds the model ID value.
	// Mutually exclusive with Model.
	ModelProperty string `json:"model_property,omitempty"`

	// Model is a static model ID value (e.g. "gpt-4", "claude-3-5-sonnet").
	// Mutually exclusive with ModelProperty.
	Model string `json:"model,omitempty"`

	// TokenTypeProperty is the meter group-by key that holds the token type.
	// Mutually exclusive with TokenType.
	TokenTypeProperty string `json:"token_type_property,omitempty"`

	// TokenType is a static token type value (e.g. "input", "output").
	// Use this when the feature tracks a single token type.
	// Mutually exclusive with TokenTypeProperty.
	TokenType string `json:"token_type,omitempty"`
}

// Validate validates the unit cost configuration.
func (u *UnitCost) Validate() error {
	if u == nil {
		return nil
	}

	switch u.Type {
	case UnitCostTypeManual:
		if u.Manual == nil {
			return errors.New("manual unit cost configuration is required when type is manual")
		}

		if u.Manual.Amount.IsNegative() {
			return errors.New("manual unit cost amount must be non-negative")
		}

		return nil

	case UnitCostTypeLLM:
		if u.LLM == nil {
			return errors.New("LLM unit cost configuration is required when type is llm")
		}

		var errs []error

		// Provider: exactly one of property or static value
		if u.LLM.ProviderProperty == "" && u.LLM.Provider == "" {
			errs = append(errs, errors.New("either provider_property or provider is required for LLM unit cost"))
		}
		if u.LLM.ProviderProperty != "" && u.LLM.Provider != "" {
			errs = append(errs, errors.New("provider_property and provider are mutually exclusive"))
		}

		// Model: exactly one of property or static value
		if u.LLM.ModelProperty == "" && u.LLM.Model == "" {
			errs = append(errs, errors.New("either model_property or model is required for LLM unit cost"))
		}
		if u.LLM.ModelProperty != "" && u.LLM.Model != "" {
			errs = append(errs, errors.New("model_property and model are mutually exclusive"))
		}

		// Token type: exactly one of property or static value
		if u.LLM.TokenTypeProperty == "" && u.LLM.TokenType == "" {
			errs = append(errs, errors.New("either token_type_property or token_type is required for LLM unit cost"))
		}
		if u.LLM.TokenTypeProperty != "" && u.LLM.TokenType != "" {
			errs = append(errs, errors.New("token_type_property and token_type are mutually exclusive"))
		}

		if u.LLM.TokenType != "" {
			validTypes := map[LLMTokenType]bool{
				LLMTokenTypeInput: true, LLMTokenTypeOutput: true,
				LLMTokenTypeInputCached: true, LLMTokenTypeReasoning: true,
				LLMTokenTypeCacheWrite: true,
			}
			if !validTypes[LLMTokenType(u.LLM.TokenType)] {
				errs = append(errs, fmt.Errorf("invalid token_type %q: expected one of input, output, input_cached, reasoning, cache_write", u.LLM.TokenType))
			}
		}

		return errors.Join(errs...)

	default:
		return fmt.Errorf("invalid unit cost type: %s", u.Type)
	}
}

// ValidateWithMeter validates that the LLM unit cost property names exist in the meter's GroupBy keys.
func (u *UnitCost) ValidateWithMeter(m meter.Meter) error {
	if u == nil || u.Type != UnitCostTypeLLM || u.LLM == nil {
		return nil
	}

	var errs []error

	if u.LLM.ProviderProperty != "" {
		if _, ok := m.GroupBy[u.LLM.ProviderProperty]; !ok {
			errs = append(errs, fmt.Errorf("provider_property %q not found in meter group-by keys", u.LLM.ProviderProperty))
		}
	}

	if u.LLM.ModelProperty != "" {
		if _, ok := m.GroupBy[u.LLM.ModelProperty]; !ok {
			errs = append(errs, fmt.Errorf("model_property %q not found in meter group-by keys", u.LLM.ModelProperty))
		}
	}

	if u.LLM.TokenTypeProperty != "" {
		if _, ok := m.GroupBy[u.LLM.TokenTypeProperty]; !ok {
			errs = append(errs, fmt.Errorf("token_type_property %q not found in meter group-by keys", u.LLM.TokenTypeProperty))
		}
	}

	return errors.Join(errs...)
}

// LLMTokenType identifies a token dimension for LLM pricing.
type LLMTokenType string

const (
	LLMTokenTypeInput       LLMTokenType = "input"
	LLMTokenTypeOutput      LLMTokenType = "output"
	LLMTokenTypeInputCached LLMTokenType = "input_cached"
	LLMTokenTypeReasoning   LLMTokenType = "reasoning"
	LLMTokenTypeCacheWrite  LLMTokenType = "cache_write"
)

// CostPerTokenForType extracts the per-token cost for the given token type from ModelPricing.
// Returns an error if the token type is unknown or the pricing has no value for the requested dimension.
func CostPerTokenForType(pricing llmcost.ModelPricing, tokenType LLMTokenType) (alpacadecimal.Decimal, error) {
	switch tokenType {
	case LLMTokenTypeInput:
		return pricing.InputPerToken, nil
	case LLMTokenTypeOutput:
		return pricing.OutputPerToken, nil
	case LLMTokenTypeInputCached:
		if pricing.InputCachedPerToken == nil {
			return alpacadecimal.Decimal{}, fmt.Errorf("no input_cached pricing available for this model")
		}
		return *pricing.InputCachedPerToken, nil
	case LLMTokenTypeReasoning:
		if pricing.ReasoningPerToken == nil {
			return alpacadecimal.Decimal{}, fmt.Errorf("no reasoning pricing available for this model")
		}
		return *pricing.ReasoningPerToken, nil
	case LLMTokenTypeCacheWrite:
		if pricing.CacheWritePerToken == nil {
			return alpacadecimal.Decimal{}, fmt.Errorf("no cache_write pricing available for this model")
		}
		return *pricing.CacheWritePerToken, nil
	default:
		return alpacadecimal.Decimal{}, fmt.Errorf("unknown LLM token type: %s", tokenType)
	}
}

// ResolveUnitCostInput is the input for resolving the per-unit cost of a feature.
type ResolveUnitCostInput struct {
	Namespace      string
	FeatureIDOrKey string
	// GroupByValues contains the meter group-by dimension values from usage data.
	// For LLM unit cost features, these are used to look up provider, model, and token type.
	GroupByValues map[string]string
}

// ResolvedUnitCost is the result of resolving a feature's per-unit cost.
type ResolvedUnitCost struct {
	// Amount is the resolved per-unit cost.
	Amount alpacadecimal.Decimal
	// Currency is the currency code (always "USD" for now).
	Currency string
}
