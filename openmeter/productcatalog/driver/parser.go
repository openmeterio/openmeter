package productcatalogdriver

import (
	"context"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/apiconverter"
	"github.com/openmeterio/openmeter/openmeter/llmcost"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/filter"
)

func MapFeatureToResponse(f feature.Feature) api.Feature {
	meterGroupByFilters := feature.ConvertMeterGroupByFiltersToMapString(f.MeterGroupByFilters)

	resp := api.Feature{
		CreatedAt:                   f.CreatedAt,
		DeletedAt:                   nil,
		UpdatedAt:                   f.UpdatedAt,
		Id:                          f.ID,
		Key:                         f.Key,
		Metadata:                    convert.MapToPointer(f.Metadata),
		Name:                        f.Name,
		ArchivedAt:                  f.ArchivedAt,
		MeterGroupByFilters:         convert.MapToPointer(meterGroupByFilters),
		AdvancedMeterGroupByFilters: convert.MapToPointer(apiconverter.ConvertStringMapToAPIPtr(f.MeterGroupByFilters)),
		MeterSlug:                   f.MeterSlug,
	}

	if f.UnitCost != nil {
		apiUnitCost := domainUnitCostToAPI(f.UnitCost)
		resp.UnitCost = &apiUnitCost
	}

	return resp
}

func MapFeatureCreateInputsRequest(namespace string, f api.FeatureCreateInputs) feature.CreateFeatureInputs {
	// if advancedMeterGroupByFilters is set, use it
	// otherwise, use legacy meterGroupByFilters
	meterGroupByFilters := lo.FromPtrOr(apiconverter.ConvertStringMapPtr(f.AdvancedMeterGroupByFilters), map[string]filter.FilterString{})
	if len(meterGroupByFilters) == 0 {
		meterGroupByFilters = feature.ConvertMapStringToMeterGroupByFilters(lo.FromPtrOr(f.MeterGroupByFilters, map[string]string{}))
	}

	inputs := feature.CreateFeatureInputs{
		Namespace:           namespace,
		Name:                f.Name,
		Key:                 f.Key,
		MeterSlug:           f.MeterSlug,
		MeterGroupByFilters: meterGroupByFilters,
		Metadata:            convert.DerefHeaderPtr[string](f.Metadata),
	}

	if f.UnitCost != nil {
		inputs.UnitCost = apiUnitCostToDomain(f.UnitCost)
	}

	return inputs
}

func domainUnitCostToAPI(u *feature.UnitCost) api.FeatureUnitCost {
	var out api.FeatureUnitCost

	switch u.Type {
	case feature.UnitCostTypeManual:
		_ = out.FromFeatureManualUnitCost(api.FeatureManualUnitCost{
			Amount: u.Manual.Amount.String(),
		})
	case feature.UnitCostTypeLLM:
		llmCost := api.FeatureLLMUnitCost{}
		if u.LLM.ProviderProperty != "" {
			llmCost.ProviderProperty = lo.ToPtr(u.LLM.ProviderProperty)
		}
		if u.LLM.Provider != "" {
			llmCost.Provider = lo.ToPtr(u.LLM.Provider)
		}
		if u.LLM.ModelProperty != "" {
			llmCost.ModelProperty = lo.ToPtr(u.LLM.ModelProperty)
		}
		if u.LLM.Model != "" {
			llmCost.Model = lo.ToPtr(u.LLM.Model)
		}
		if u.LLM.TokenTypeProperty != "" {
			llmCost.TokenTypeProperty = lo.ToPtr(u.LLM.TokenTypeProperty)
		}
		if u.LLM.TokenType != "" {
			llmCost.TokenType = lo.ToPtr(u.LLM.TokenType)
		}
		_ = out.FromFeatureLLMUnitCost(llmCost)
	}

	return out
}

func apiUnitCostToDomain(u *api.FeatureUnitCost) *feature.UnitCost {
	discriminator, err := u.Discriminator()
	if err != nil {
		return nil
	}

	switch discriminator {
	case "manual":
		manual, err := u.AsFeatureManualUnitCost()
		if err != nil {
			return nil
		}

		amount, _ := alpacadecimal.NewFromString(string(manual.Amount))

		return &feature.UnitCost{
			Type: feature.UnitCostTypeManual,
			Manual: &feature.ManualUnitCost{
				Amount: amount,
			},
		}
	case "llm":
		llm, err := u.AsFeatureLLMUnitCost()
		if err != nil {
			return nil
		}

		return &feature.UnitCost{
			Type: feature.UnitCostTypeLLM,
			LLM: &feature.LLMUnitCost{
				ProviderProperty:  lo.FromPtrOr(llm.ProviderProperty, ""),
				Provider:          lo.FromPtrOr(llm.Provider, ""),
				ModelProperty:     lo.FromPtrOr(llm.ModelProperty, ""),
				Model:             lo.FromPtrOr(llm.Model, ""),
				TokenTypeProperty: lo.FromPtrOr(llm.TokenTypeProperty, ""),
				TokenType:         lo.FromPtrOr(llm.TokenType, ""),
			},
		}
	}

	return nil
}

// resolveLLMPricing extracts provider and model from the feature's meterGroupByFilters
// and resolves the current pricing from the LLM cost database.
// Returns nil if provider/model can't be determined or pricing can't be resolved.
func resolveLLMPricing(ctx context.Context, svc llmcost.Service, feat *feature.Feature) *llmcost.ModelPricing {
	if feat.UnitCost == nil || feat.UnitCost.LLM == nil {
		return nil
	}

	llmConf := feat.UnitCost.LLM

	// Resolve provider: static value or from meterGroupByFilters
	provider := llmConf.Provider
	if provider == "" {
		provider = extractEqFilterValue(feat.MeterGroupByFilters, llmConf.ProviderProperty)
	}
	if provider == "" {
		return nil
	}

	// Resolve model: static value or from meterGroupByFilters
	model := llmConf.Model
	if model == "" {
		model = extractEqFilterValue(feat.MeterGroupByFilters, llmConf.ModelProperty)
	}
	if model == "" {
		return nil
	}

	price, err := svc.ResolvePrice(ctx, llmcost.ResolvePriceInput{
		Namespace: feat.Namespace,
		Provider:  llmcost.Provider(provider),
		ModelID:   model,
	})
	if err != nil {
		return nil
	}

	return &price.Pricing
}

// extractEqFilterValue extracts a simple $eq value from a MeterGroupByFilters map for the given key.
func extractEqFilterValue(filters feature.MeterGroupByFilters, key string) string {
	if filters == nil {
		return ""
	}

	f, ok := filters[key]
	if !ok || f.Eq == nil {
		return ""
	}

	return *f.Eq
}

// enrichFeatureResponseWithPricing adds resolved LLM pricing to the feature API response.
func enrichFeatureResponseWithPricing(resp *api.Feature, pricing *llmcost.ModelPricing) {
	if resp.UnitCost == nil || pricing == nil {
		return
	}

	llmCost, err := resp.UnitCost.AsFeatureLLMUnitCost()
	if err != nil {
		return
	}

	apiPricing := api.FeatureLLMUnitCostPricing{
		InputPerToken:  api.Numeric(pricing.InputPerToken.String()),
		OutputPerToken: api.Numeric(pricing.OutputPerToken.String()),
	}

	if pricing.InputCachedPerToken != nil {
		v := api.Numeric(pricing.InputCachedPerToken.String())
		apiPricing.InputCachedPerToken = &v
	}

	if pricing.ReasoningPerToken != nil {
		v := api.Numeric(pricing.ReasoningPerToken.String())
		apiPricing.ReasoningPerToken = &v
	}

	if pricing.CacheWritePerToken != nil {
		v := api.Numeric(pricing.CacheWritePerToken.String())
		apiPricing.CacheWritePerToken = &v
	}

	llmCost.Pricing = &apiPricing
	_ = resp.UnitCost.FromFeatureLLMUnitCost(llmCost)
}
