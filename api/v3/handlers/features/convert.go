package features

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oapi-codegen/nullable"
	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/labels"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/llmcost"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/filter"
)

func convertFeatureToAPI(f feature.Feature) (api.Feature, error) {
	resp := api.Feature{
		Id:          f.ID,
		Key:         f.Key,
		Name:        f.Name,
		Description: f.Description,
		Labels:      labels.FromMetadata(f.Metadata),
		CreatedAt:   f.CreatedAt,
		UpdatedAt:   f.UpdatedAt,
		DeletedAt:   f.ArchivedAt,
	}

	if f.MeterID != nil {
		resp.Meter = &api.FeatureMeterReference{
			Id: *f.MeterID,
		}

		if len(f.MeterGroupByFilters) > 0 {
			filters := convertFiltersToAPI(f.MeterGroupByFilters)
			resp.Meter.Filters = &filters
		}
	}

	if f.UnitCost != nil {
		apiUnitCost, err := convertUnitCostToAPI(f.UnitCost)
		if err != nil {
			return api.Feature{}, fmt.Errorf("failed to convert unit cost: %w", err)
		}
		resp.UnitCost = &apiUnitCost
	}

	return resp, nil
}

func convertCreateRequestToDomain(ns string, body api.CreateFeatureRequest, meterID *string) (feature.CreateFeatureInputs, error) {
	metadata, err := labels.ToMetadata(body.Labels)
	if err != nil {
		return feature.CreateFeatureInputs{}, fmt.Errorf("failed to convert labels: %w", err)
	}

	inputs := feature.CreateFeatureInputs{
		Namespace:   ns,
		Name:        body.Name,
		Description: body.Description,
		Key:         body.Key,
		MeterID:     meterID,
		Metadata:    metadata,
	}

	if body.Meter != nil {
		if body.Meter.Filters != nil {
			inputs.MeterGroupByFilters = convertFiltersFromAPI(*body.Meter.Filters)
		}
	}

	if body.UnitCost != nil {
		unitCost, err := convertUnitCostFromAPI(body.UnitCost)
		if err != nil {
			return feature.CreateFeatureInputs{}, fmt.Errorf("invalid unit cost: %w", err)
		}
		inputs.UnitCost = unitCost
	}

	return inputs, nil
}

func convertUpdateRequestToDomain(ns string, featureID string, body api.UpdateFeatureRequest) (feature.UpdateFeatureInputs, error) {
	input := feature.UpdateFeatureInputs{
		Namespace: ns,
		ID:        featureID,
	}

	if body.UnitCost.IsNull() {
		input.UnitCost = nullable.NewNullNullable[feature.UnitCost]()
	} else if body.UnitCost.IsSpecified() {
		v, err := body.UnitCost.Get()
		if err != nil {
			return feature.UpdateFeatureInputs{}, fmt.Errorf("invalid unit cost: %w", err)
		}
		unitCost, err := convertUnitCostFromAPI(&v)
		if err != nil {
			return feature.UpdateFeatureInputs{}, fmt.Errorf("invalid unit cost: %w", err)
		}
		input.UnitCost = nullable.NewNullableWithValue(*unitCost)
	}

	return input, nil
}

func convertUnitCostToAPI(u *feature.UnitCost) (api.BillingFeatureUnitCost, error) {
	var out api.BillingFeatureUnitCost

	switch u.Type {
	case feature.UnitCostTypeManual:
		if err := out.FromBillingFeatureManualUnitCost(api.BillingFeatureManualUnitCost{
			Amount: u.Manual.Amount.String(),
		}); err != nil {
			return out, fmt.Errorf("failed to convert manual unit cost: %w", err)
		}
	case feature.UnitCostTypeLLM:
		llmCost := api.BillingFeatureLLMUnitCost{}
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
			llmCost.TokenType = lo.ToPtr(api.BillingFeatureLLMTokenType(u.LLM.TokenType))
		}
		if err := out.FromBillingFeatureLLMUnitCost(llmCost); err != nil {
			return out, fmt.Errorf("failed to convert LLM unit cost: %w", err)
		}
	default:
		return out, fmt.Errorf("unknown unit cost type: %s", u.Type)
	}

	return out, nil
}

func convertUnitCostFromAPI(u *api.BillingFeatureUnitCost) (*feature.UnitCost, error) {
	discriminator, err := u.Discriminator()
	if err != nil {
		return nil, fmt.Errorf("failed to determine unit cost type: %w", err)
	}

	switch discriminator {
	case "manual":
		manual, err := u.AsBillingFeatureManualUnitCost()
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
		llm, err := u.AsBillingFeatureLLMUnitCost()
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
				TokenType:         string(lo.FromPtrOr(llm.TokenType, "")),
			},
		}, nil
	default:
		return nil, fmt.Errorf("unknown unit cost type: %s", discriminator)
	}
}

func enrichFeatureResponseWithPricing(resp *api.Feature, pricing *llmcost.ModelPricing) {
	if resp.UnitCost == nil || pricing == nil {
		return
	}

	disc, err := resp.UnitCost.Discriminator()
	if err != nil || disc != "llm" {
		return
	}

	llmCost, err := resp.UnitCost.AsBillingFeatureLLMUnitCost()
	if err != nil {
		return
	}

	apiPricing := api.BillingFeatureLLMUnitCostPricing{
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
	_ = resp.UnitCost.FromBillingFeatureLLMUnitCost(llmCost)
}

// resolveLLMPricing resolves LLM pricing for a feature from the LLM cost database.
func resolveLLMPricing(ctx context.Context, svc llmcost.Service, feat *feature.Feature) *llmcost.ModelPricing {
	if feat.UnitCost == nil || feat.UnitCost.LLM == nil {
		return nil
	}

	llmConf := feat.UnitCost.LLM

	provider := llmConf.Provider
	if provider == "" {
		provider = extractEqFilterValue(feat.MeterGroupByFilters, llmConf.ProviderProperty)
	}
	if provider == "" {
		return nil
	}

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

func convertFiltersFromAPI(apiFilters map[string]api.QueryFilterStringMapItem) feature.MeterGroupByFilters {
	result := make(feature.MeterGroupByFilters, len(apiFilters))
	for k, v := range apiFilters {
		result[k] = request.ConvertQueryFilterStringMapItem(v)
	}
	return result
}

func convertFiltersToAPI(filters feature.MeterGroupByFilters) map[string]api.QueryFilterStringMapItem {
	result := make(map[string]api.QueryFilterStringMapItem, len(filters))
	for k, v := range filters {
		result[k] = convertFilterStringToAPIMapItem(v)
	}
	return result
}

func convertFilterStringToAPIMapItem(f filter.FilterString) api.QueryFilterStringMapItem {
	return api.QueryFilterStringMapItem{
		Exists:    f.Exists,
		Eq:        f.Eq,
		Neq:       f.Ne,
		In:        f.In,
		Nin:       f.Nin,
		Contains:  filter.ReverseContainsPattern(f.Like),
		Ncontains: filter.ReverseContainsPattern(f.Nlike),
		And:       convertFilterStringListToAPI(f.And),
		Or:        convertFilterStringListToAPI(f.Or),
	}
}

func convertFilterStringListToAPI(filters *[]filter.FilterString) *[]api.QueryFilterString {
	if filters == nil {
		return nil
	}
	result := make([]api.QueryFilterString, len(*filters))
	for i, f := range *filters {
		result[i] = convertFilterStringToAPIQueryFilter(f)
	}
	return &result
}

func convertFilterStringToAPIQueryFilter(f filter.FilterString) api.QueryFilterString {
	return api.QueryFilterString{
		Eq:        f.Eq,
		Neq:       f.Ne,
		In:        f.In,
		Nin:       f.Nin,
		Contains:  filter.ReverseContainsPattern(f.Like),
		Ncontains: filter.ReverseContainsPattern(f.Nlike),
		And:       convertFilterStringListToAPI(f.And),
		Or:        convertFilterStringListToAPI(f.Or),
	}
}
