package llmcost

import (
	"fmt"
	"strings"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/filters"
	"github.com/openmeterio/openmeter/openmeter/llmcost"
	"github.com/openmeterio/openmeter/pkg/models"
)

// providerDisplayNames maps well-known provider IDs to their formatted display names.
var providerDisplayNames = map[string]string{
	"anthropic":  "Anthropic",
	"azure":      "Azure",
	"bedrock":    "Amazon Bedrock",
	"cohere":     "Cohere",
	"gemini":     "Google Gemini",
	"huggingface": "HuggingFace",
	"llama2":     "Llama",
	"mistral":    "Mistral",
	"openai":     "OpenAI",
	"vertex":     "Google Vertex",
}

// formatProviderName returns the display name for a provider ID.
// Known providers get their canonical display name; unknown providers
// get their ID with the first letter capitalized.
func formatProviderName(id string) string {
	if name, ok := providerDisplayNames[strings.ToLower(id)]; ok {
		return name
	}

	if id == "" {
		return ""
	}

	// Fallback: capitalize first letter
	return strings.ToUpper(id[:1]) + id[1:]
}

func domainPriceToAPI(p llmcost.Price) api.LLMCostPrice {
	// Map internal source to API source: manual stays manual, everything else is system.
	source := api.LLMCostPriceSourceSystem
	if p.Source == llmcost.PriceSourceManual {
		source = api.LLMCostPriceSourceManual
	}

	providerID := string(p.Provider)

	out := api.LLMCostPrice{
		Id: p.ID,
		Provider: api.LLMCostProvider{
			Id:   providerID,
			Name: formatProviderName(providerID),
		},
		Model: api.LLMCostModel{
			Id:   p.ModelID,
			Name: p.ModelName,
		},
		Currency:      p.Currency,
		Source:        source,
		EffectiveFrom: p.EffectiveFrom,
		EffectiveTo:   p.EffectiveTo,
		Pricing:       domainPricingToAPI(p.Pricing),
	}

	if !p.CreatedAt.IsZero() {
		out.CreatedAt = &p.CreatedAt
	}

	if !p.UpdatedAt.IsZero() {
		out.UpdatedAt = &p.UpdatedAt
	}

	return out
}

func domainPricingToAPI(p llmcost.ModelPricing) api.LLMCostModelPricing {
	out := api.LLMCostModelPricing{
		InputPerToken:  p.InputPerToken.String(),
		OutputPerToken: p.OutputPerToken.String(),
	}

	if p.CacheReadPerToken != nil {
		out.CacheReadPerToken = lo.ToPtr(p.CacheReadPerToken.String())
	}

	if p.CacheWritePerToken != nil {
		out.CacheWritePerToken = lo.ToPtr(p.CacheWritePerToken.String())
	}

	if p.ReasoningPerToken != nil {
		out.ReasoningPerToken = lo.ToPtr(p.ReasoningPerToken.String())
	}

	return out
}

func apiPricingToDomain(p api.LLMCostModelPricing) (llmcost.ModelPricing, error) {
	inputPerToken, err := decimalFromString(p.InputPerToken)
	if err != nil {
		return llmcost.ModelPricing{}, models.NewGenericValidationError(
			fmt.Errorf("invalid input_per_token: %w", err),
		)
	}

	outputPerToken, err := decimalFromString(p.OutputPerToken)
	if err != nil {
		return llmcost.ModelPricing{}, models.NewGenericValidationError(
			fmt.Errorf("invalid output_per_token: %w", err),
		)
	}

	out := llmcost.ModelPricing{
		InputPerToken:  inputPerToken,
		OutputPerToken: outputPerToken,
	}

	if p.CacheReadPerToken != nil {
		d, err := decimalFromString(*p.CacheReadPerToken)
		if err != nil {
			return llmcost.ModelPricing{}, models.NewGenericValidationError(
				fmt.Errorf("invalid cache_read_per_token: %w", err),
			)
		}
		out.CacheReadPerToken = &d
	}

	if p.ReasoningPerToken != nil {
		d, err := decimalFromString(*p.ReasoningPerToken)
		if err != nil {
			return llmcost.ModelPricing{}, models.NewGenericValidationError(
				fmt.Errorf("invalid reasoning_per_token: %w", err),
			)
		}
		out.ReasoningPerToken = &d
	}

	if p.CacheWritePerToken != nil {
		d, err := decimalFromString(*p.CacheWritePerToken)
		if err != nil {
			return llmcost.ModelPricing{}, models.NewGenericValidationError(
				fmt.Errorf("invalid cache_write_per_token: %w", err),
			)
		}
		out.CacheWritePerToken = &d
	}

	return out, nil
}

func apiCreateOverrideToDomain(ns string, body api.LLMCostOverrideCreate) (llmcost.CreateOverrideInput, error) {
	pricing, err := apiPricingToDomain(body.Pricing)
	if err != nil {
		return llmcost.CreateOverrideInput{}, err
	}

	input := llmcost.CreateOverrideInput{
		Namespace:     ns,
		Provider:      llmcost.Provider(body.Provider),
		ModelID:       body.ModelId,
		Pricing:       pricing,
		Currency:      body.Currency,
		EffectiveFrom: body.EffectiveFrom,
	}

	if body.ModelName != nil {
		input.ModelName = *body.ModelName
	}

	if body.EffectiveTo != nil {
		input.EffectiveTo = body.EffectiveTo
	}

	return input, nil
}

func decimalFromString(s string) (alpacadecimal.Decimal, error) {
	v, err := alpacadecimal.NewFromString(s)
	if err != nil {
		return alpacadecimal.Decimal{}, models.NewGenericValidationError(
			fmt.Errorf("invalid decimal: %w", err),
		)
	}

	return v, nil
}

// filterSingleStringToDomain converts an API FilterSingleString to the domain StringFilter.
// Returns nil if the input is nil or empty.
func filterSingleStringToDomain(f *api.FilterSingleString) (*filters.StringFilter, error) {
	if f == nil {
		return nil, nil
	}

	out := &filters.StringFilter{
		Eq:       f.Eq,
		Neq:      f.Neq,
		Contains: f.Contains,
	}

	if err := out.Validate(); err != nil {
		return nil, models.NewGenericValidationError(err)
	}

	if out.IsEmpty() {
		return nil, nil
	}

	return out, nil
}
