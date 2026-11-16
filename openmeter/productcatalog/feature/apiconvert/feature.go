package apiconvert

import (
	"fmt"

	"github.com/samber/lo"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/apiconverter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/convert"
)

func ToAPIFeature(f feature.Feature) (apiv1models.Feature, error) {
	meterGroupByFilters := feature.ConvertMeterGroupByFiltersToMapString(f.MeterGroupByFilters)

	resp := apiv1models.Feature{
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
			return apiv1models.Feature{}, fmt.Errorf("failed to convert unit cost: %w", err)
		}

		resp.UnitCost = &apiUnitCost
	}

	return resp, nil
}

func toAPIFeatureUnitCost(u *feature.UnitCost) (apiv1models.FeatureUnitCost, error) {
	var out apiv1models.FeatureUnitCost

	switch u.Type {
	case feature.UnitCostTypeManual:
		if err := out.FromFeatureManualUnitCost(apiv1models.FeatureManualUnitCost{
			Amount: u.Manual.Amount.String(),
		}); err != nil {
			return out, fmt.Errorf("failed to convert manual unit cost: %w", err)
		}
	case feature.UnitCostTypeLLM:
		llmCost := apiv1models.FeatureLLMUnitCost{}
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
