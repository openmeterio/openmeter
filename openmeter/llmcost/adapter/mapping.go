package adapter

import (
	"errors"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/llmcost"
	"github.com/openmeterio/openmeter/pkg/models"
)

func mapPriceFromEntity(entity *db.LLMCostPrice) (llmcost.Price, error) {
	if entity == nil {
		return llmcost.Price{}, errors.New("entity is required")
	}

	price := llmcost.Price{
		ManagedModel: models.ManagedModel{
			CreatedAt: entity.CreatedAt,
			UpdatedAt: entity.UpdatedAt,
			DeletedAt: entity.DeletedAt,
		},
		ID:        entity.ID,
		Namespace: entity.Namespace,
		Provider:  llmcost.Provider(entity.Provider),
		ModelID:   entity.ModelID,
		ModelName: entity.ModelName,
		Pricing: llmcost.ModelPricing{
			InputPerToken:  entity.InputPerToken,
			OutputPerToken: entity.OutputPerToken,
		},
		Currency:      entity.Currency,
		Source:        llmcost.PriceSource(entity.Source),
		SourcePrices:  entity.SourcePrices,
		EffectiveFrom: entity.EffectiveFrom,
		EffectiveTo:   entity.EffectiveTo,
		Metadata:      models.NewMetadata(entity.Metadata),
	}

	// Map optional pricing fields (zero value means not set)
	if !entity.InputCachedPerToken.IsZero() {
		price.Pricing.InputCachedPerToken = lo.ToPtr(entity.InputCachedPerToken)
	}

	if !entity.ReasoningPerToken.IsZero() {
		price.Pricing.ReasoningPerToken = lo.ToPtr(entity.ReasoningPerToken)
	}

	if !entity.CacheWritePerToken.IsZero() {
		price.Pricing.CacheWritePerToken = lo.ToPtr(entity.CacheWritePerToken)
	}

	return price, nil
}

// decimalOrZero returns the decimal value or zero if the pointer is nil.
func decimalOrZero(d *alpacadecimal.Decimal) alpacadecimal.Decimal {
	if d == nil {
		return alpacadecimal.Decimal{}
	}

	return *d
}
