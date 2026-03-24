package productcatalogdriver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/apiconverter"
	"github.com/openmeterio/openmeter/openmeter/llmcost"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	productcataloghttp "github.com/openmeterio/openmeter/openmeter/productcatalog/http"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/filter"
)

func MapFeatureToResponse(f feature.Feature) (api.Feature, error) {
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
		apiUnitCost, err := domainUnitCostToAPI(f.UnitCost)
		if err != nil {
			return api.Feature{}, fmt.Errorf("failed to convert unit cost: %w", err)
		}
		resp.UnitCost = &apiUnitCost
	}

	if len(f.UnitConfig) > 0 {
		var uc productcatalog.UnitConfig
		if err := json.Unmarshal(f.UnitConfig, &uc); err == nil {
			apiUC := productcataloghttp.FromUnitConfig(uc)
			resp.UnitConfig = &apiUC
		}
	}

	return resp, nil
}

func MapFeatureCreateInputsRequest(namespace string, f api.FeatureCreateInputs, meterID *string) (feature.CreateFeatureInputs, error) {
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
		MeterID:             meterID,
		MeterGroupByFilters: meterGroupByFilters,
		Metadata:            convert.DerefHeaderPtr[string](f.Metadata),
	}

	if f.UnitCost != nil {
		unitCost, err := apiUnitCostToDomain(f.UnitCost)
		if err != nil {
			return feature.CreateFeatureInputs{}, fmt.Errorf("invalid unit cost: %w", err)
		}
		inputs.UnitCost = unitCost
	}

	if f.UnitConfig != nil {
		uc, err := productcataloghttp.AsUnitConfig(*f.UnitConfig)
		if err != nil {
			return feature.CreateFeatureInputs{}, fmt.Errorf("invalid unit config: %w", err)
		}
		if err := uc.Validate(); err != nil {
			return feature.CreateFeatureInputs{}, fmt.Errorf("invalid unit config: %w", err)
		}
		ucJSON, err := json.Marshal(uc)
		if err != nil {
			return feature.CreateFeatureInputs{}, fmt.Errorf("failed to marshal unit config: %w", err)
		}
		inputs.UnitConfig = ucJSON
	}

	return inputs, nil
}

func domainUnitCostToAPI(u *feature.UnitCost) (api.FeatureUnitCost, error) {
	var out api.FeatureUnitCost

	switch u.Type {
	case feature.UnitCostTypeManual:
		if err := out.FromFeatureManualUnitCost(api.FeatureManualUnitCost{
			Amount: u.Manual.Amount.String(),
		}); err != nil {
			return out, fmt.Errorf("failed to convert manual unit cost: %w", err)
		}
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
		if err := out.FromFeatureLLMUnitCost(llmCost); err != nil {
			return out, fmt.Errorf("failed to convert LLM unit cost: %w", err)
		}
	default:
		return out, fmt.Errorf("unknown unit cost type: %s", u.Type)
	}

	return out, nil
}

func apiUnitCostToDomain(u *api.FeatureUnitCost) (*feature.UnitCost, error) {
	discriminator, err := u.Discriminator()
	if err != nil {
		return nil, fmt.Errorf("failed to determine unit cost type: %w", err)
	}

	switch discriminator {
	case "manual":
		manual, err := u.AsFeatureManualUnitCost()
		if err != nil {
			return nil, fmt.Errorf("failed to parse manual unit cost: %w", err)
		}

		amount, err := alpacadecimal.NewFromString(manual.Amount)
		if err != nil {
			return nil, fmt.Errorf("invalid manual unit cost amount %q: %w", manual.Amount, err)
		}

		return &feature.UnitCost{
			Type: feature.UnitCostTypeManual,
			Manual: &feature.ManualUnitCost{
				Amount: amount,
			},
		}, nil
	case "llm":
		llm, err := u.AsFeatureLLMUnitCost()
		if err != nil {
			return nil, fmt.Errorf("failed to parse LLM unit cost: %w", err)
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
		}, nil
	default:
		return nil, fmt.Errorf("unknown unit cost type: %s", discriminator)
	}
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
		InputPerToken:  pricing.InputPerToken.String(),
		OutputPerToken: pricing.OutputPerToken.String(),
	}

	if pricing.CacheReadPerToken != nil {
		v := pricing.CacheReadPerToken.String()
		apiPricing.CacheReadPerToken = &v
	}

	if pricing.CacheWritePerToken != nil {
		v := pricing.CacheWritePerToken.String()
		apiPricing.CacheWritePerToken = &v
	}

	if pricing.ReasoningPerToken != nil {
		v := pricing.ReasoningPerToken.String()
		apiPricing.ReasoningPerToken = &v
	}

	llmCost.Pricing = &apiPricing
	_ = resp.UnitCost.FromFeatureLLMUnitCost(llmCost)
}
