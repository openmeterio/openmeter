package productcatalogdriver

import (
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/apiconverter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
)

func MapFeatureToResponse(f feature.Feature) api.Feature {
	meterGroupByFilters := feature.ConvertMeterGroupByFiltersToMapString(f.MeterGroupByFilters)

	feature := api.Feature{
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

	if f.Cost != nil {
		feature.Cost = lo.ToPtr(MapCostToResponse(*f.Cost))
	}

	return feature
}

func MapCostToResponse(cost feature.Cost) api.Cost {
	return api.Cost{
		Kind:          api.CostKind(cost.Kind),
		Currency:      cost.Currency.String(),
		PerUnitAmount: cost.PerUnitAmount.String(),
		ProviderId:    cost.ProviderID,
	}
}

func MapFeatureCreateInputsRequest(namespace string, f api.FeatureCreateInputs) (feature.CreateFeatureInputs, error) {
	// if advancedMeterGroupByFilters is set, use it
	// otherwise, use legacy meterGroupByFilters
	meterGroupByFilters := lo.FromPtrOr(apiconverter.ConvertStringMapPtr(f.AdvancedMeterGroupByFilters), map[string]filter.FilterString{})
	if len(meterGroupByFilters) == 0 {
		meterGroupByFilters = feature.ConvertMapStringToMeterGroupByFilters(lo.FromPtrOr(f.MeterGroupByFilters, map[string]string{}))
	}

	createInput := feature.CreateFeatureInputs{
		Namespace:           namespace,
		Name:                f.Name,
		Key:                 f.Key,
		MeterSlug:           f.MeterSlug,
		MeterGroupByFilters: meterGroupByFilters,
		Metadata:            convert.DerefHeaderPtr[string](f.Metadata),
	}

	// Map cost
	if f.Cost != nil {
		costInput := feature.CostMutateInput{
			Kind:       feature.CostKind(f.Cost.Kind),
			Currency:   currency.Code(f.Cost.Currency),
			ProviderID: f.Cost.ProviderId,
		}

		if f.Cost.PerUnitAmount != nil {
			perUnitAmount, err := alpacadecimal.NewFromString(*f.Cost.PerUnitAmount)
			if err != nil {
				return createInput, models.NewGenericValidationError(
					fmt.Errorf("invalid per unit amount: %w", err),
				)
			}
			costInput.PerUnitAmount = &perUnitAmount
		}

		createInput.Cost = &costInput

		err := costInput.Validate()
		if err != nil {
			return createInput, models.NewGenericValidationError(
				fmt.Errorf("invalid cost input: %w", err),
			)
		}
	}

	return createInput, nil
}
