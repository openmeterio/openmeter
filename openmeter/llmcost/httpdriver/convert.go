package httpdriver

import (
	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/llmcost"
)

func domainPriceToAPI(p llmcost.Price) api.LLMCostPrice {
	// Map internal source to API source: manual stays manual, everything else is system.
	source := api.LLMCostPriceSourceSystem
	if p.Source == llmcost.PriceSourceManual {
		source = api.LLMCostPriceSourceManual
	}

	out := api.LLMCostPrice{
		Id:            p.ID,
		Provider:      string(p.Provider),
		ModelId:       p.ModelID,
		ModelName:     p.ModelName,
		Currency:      p.Currency,
		Source:        source,
		EffectiveFrom: p.EffectiveFrom,
		Pricing:       domainPricingToAPI(p.Pricing),
	}

	if p.EffectiveTo != nil {
		out.EffectiveTo = p.EffectiveTo
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

	if p.InputCachedPerToken != nil {
		out.InputCachedPerToken = lo.ToPtr(p.InputCachedPerToken.String())
	}

	if p.ReasoningPerToken != nil {
		out.ReasoningPerToken = lo.ToPtr(p.ReasoningPerToken.String())
	}

	if p.CacheWritePerToken != nil {
		out.CacheWritePerToken = lo.ToPtr(p.CacheWritePerToken.String())
	}

	return out
}

func apiPricingToDomain(p api.LLMCostModelPricing) llmcost.ModelPricing {
	out := llmcost.ModelPricing{
		InputPerToken:  decimalFromString(p.InputPerToken),
		OutputPerToken: decimalFromString(p.OutputPerToken),
	}

	if p.InputCachedPerToken != nil {
		d := decimalFromString(*p.InputCachedPerToken)
		out.InputCachedPerToken = &d
	}

	if p.ReasoningPerToken != nil {
		d := decimalFromString(*p.ReasoningPerToken)
		out.ReasoningPerToken = &d
	}

	if p.CacheWritePerToken != nil {
		d := decimalFromString(*p.CacheWritePerToken)
		out.CacheWritePerToken = &d
	}

	return out
}

func apiCreateOverrideToDomain(ns string, body api.LLMCostOverrideCreate) llmcost.CreateOverrideInput {
	input := llmcost.CreateOverrideInput{
		Namespace:     ns,
		Provider:      llmcost.Provider(body.Provider),
		ModelID:       body.ModelId,
		Pricing:       apiPricingToDomain(body.Pricing),
		EffectiveFrom: body.EffectiveFrom,
	}

	if body.ModelName != nil {
		input.ModelName = *body.ModelName
	}

	if body.EffectiveTo != nil {
		input.EffectiveTo = body.EffectiveTo
	}

	return input
}

func apiUpdateOverrideToDomain(ns string, id string, body api.LLMCostOverrideUpdate) llmcost.UpdateOverrideInput {
	input := llmcost.UpdateOverrideInput{
		ID:        id,
		Namespace: ns,
	}

	if body.Pricing != nil {
		input.Pricing = apiPricingToDomain(*body.Pricing)
	}

	if body.EffectiveTo != nil {
		input.EffectiveTo = body.EffectiveTo
	}

	return input
}

func decimalFromString(s string) alpacadecimal.Decimal {
	d, _ := alpacadecimal.NewFromString(s)
	return d
}
