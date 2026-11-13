package apiconvert

import (
	"fmt"

	"github.com/samber/lo"

	apimodels "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/apiconverter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/convert"
)

func ToAPIFeature(f feature.Feature) (apimodels.Feature, error) {
	meterGroupByFilters := feature.ConvertMeterGroupByFiltersToMapString(f.MeterGroupByFilters)

	resp := apimodels.Feature{
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
		apiUnitCost, err := toAPIFeatureUnitCost(f.UnitCost)
		if err != nil {
			return apimodels.Feature{}, fmt.Errorf("failed to convert unit cost: %w", err)
		}

		resp.UnitCost = &apiUnitCost
	}

	return resp, nil
}

func toAPIFeatureUnitCost(u *feature.UnitCost) (apimodels.FeatureUnitCost, error) {
	var out apimodels.FeatureUnitCost

	switch u.Type {
	case feature.UnitCostTypeManual:
		if err := out.FromFeatureManualUnitCost(apimodels.FeatureManualUnitCost{
			Amount: u.Manual.Amount.String(),
		}); err != nil {
			return out, fmt.Errorf("failed to convert manual unit cost: %w", err)
		}
	case feature.UnitCostTypeLLM:
		llmCost := apimodels.FeatureLLMUnitCost{}
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
