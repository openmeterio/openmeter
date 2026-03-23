package features

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/llmcost"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/filter"
)

func TestConvertUnitCostToAPI(t *testing.T) {
	t.Run("manual unit cost", func(t *testing.T) {
		uc := &feature.UnitCost{
			Type: feature.UnitCostTypeManual,
			Manual: &feature.ManualUnitCost{
				Amount: alpacadecimal.NewFromFloat(0.005),
			},
		}

		result, err := convertUnitCostToAPI(uc)
		require.NoError(t, err)

		disc, err := result.Discriminator()
		require.NoError(t, err)
		assert.Equal(t, "manual", disc)

		manual, err := result.AsBillingFeatureManualUnitCost()
		require.NoError(t, err)
		assert.Equal(t, api.Numeric("0.005"), manual.Amount)
	})

	t.Run("llm unit cost with properties", func(t *testing.T) {
		uc := &feature.UnitCost{
			Type: feature.UnitCostTypeLLM,
			LLM: &feature.LLMUnitCost{
				ProviderProperty:  "provider",
				ModelProperty:     "model",
				TokenTypeProperty: "type",
			},
		}

		result, err := convertUnitCostToAPI(uc)
		require.NoError(t, err)

		disc, err := result.Discriminator()
		require.NoError(t, err)
		assert.Equal(t, "llm", disc)

		llm, err := result.AsBillingFeatureLLMUnitCost()
		require.NoError(t, err)
		assert.Equal(t, lo.ToPtr("provider"), llm.ProviderProperty)
		assert.Equal(t, lo.ToPtr("model"), llm.ModelProperty)
		assert.Equal(t, lo.ToPtr("type"), llm.TokenTypeProperty)
		assert.Nil(t, llm.Provider)
		assert.Nil(t, llm.Model)
		assert.Nil(t, llm.TokenType)
	})

	t.Run("llm unit cost with static values", func(t *testing.T) {
		uc := &feature.UnitCost{
			Type: feature.UnitCostTypeLLM,
			LLM: &feature.LLMUnitCost{
				Provider:  "openai",
				Model:     "gpt-4",
				TokenType: "input",
			},
		}

		result, err := convertUnitCostToAPI(uc)
		require.NoError(t, err)

		llm, err := result.AsBillingFeatureLLMUnitCost()
		require.NoError(t, err)
		assert.Equal(t, lo.ToPtr("openai"), llm.Provider)
		assert.Equal(t, lo.ToPtr("gpt-4"), llm.Model)
		assert.Equal(t, lo.ToPtr(api.BillingFeatureLLMTokenTypeInput), llm.TokenType)
		assert.Nil(t, llm.ProviderProperty)
		assert.Nil(t, llm.ModelProperty)
		assert.Nil(t, llm.TokenTypeProperty)
	})

	t.Run("unknown type returns error", func(t *testing.T) {
		uc := &feature.UnitCost{Type: "unknown"}
		_, err := convertUnitCostToAPI(uc)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown unit cost type")
	})
}

func TestConvertUnitCostFromAPI(t *testing.T) {
	t.Run("manual unit cost", func(t *testing.T) {
		var apiUC api.BillingFeatureUnitCost
		err := apiUC.FromBillingFeatureManualUnitCost(api.BillingFeatureManualUnitCost{
			Amount: "0.123",
		})
		require.NoError(t, err)

		result, err := convertUnitCostFromAPI(&apiUC)
		require.NoError(t, err)
		assert.Equal(t, feature.UnitCostTypeManual, result.Type)
		assert.NotNil(t, result.Manual)
		assert.Equal(t, "0.123", result.Manual.Amount.String())
		assert.Nil(t, result.LLM)
	})

	t.Run("llm unit cost with properties", func(t *testing.T) {
		var apiUC api.BillingFeatureUnitCost
		err := apiUC.FromBillingFeatureLLMUnitCost(api.BillingFeatureLLMUnitCost{
			ProviderProperty:  lo.ToPtr("provider"),
			ModelProperty:     lo.ToPtr("model"),
			TokenTypeProperty: lo.ToPtr("type"),
		})
		require.NoError(t, err)

		result, err := convertUnitCostFromAPI(&apiUC)
		require.NoError(t, err)
		assert.Equal(t, feature.UnitCostTypeLLM, result.Type)
		assert.NotNil(t, result.LLM)
		assert.Equal(t, "provider", result.LLM.ProviderProperty)
		assert.Equal(t, "model", result.LLM.ModelProperty)
		assert.Equal(t, "type", result.LLM.TokenTypeProperty)
		assert.Empty(t, result.LLM.Provider)
		assert.Empty(t, result.LLM.Model)
		assert.Empty(t, result.LLM.TokenType)
	})

	t.Run("llm unit cost with static values", func(t *testing.T) {
		var apiUC api.BillingFeatureUnitCost
		err := apiUC.FromBillingFeatureLLMUnitCost(api.BillingFeatureLLMUnitCost{
			Provider:  lo.ToPtr("anthropic"),
			Model:     lo.ToPtr("claude-3-5-sonnet"),
			TokenType: lo.ToPtr(api.BillingFeatureLLMTokenTypeOutput),
		})
		require.NoError(t, err)

		result, err := convertUnitCostFromAPI(&apiUC)
		require.NoError(t, err)
		assert.Equal(t, feature.UnitCostTypeLLM, result.Type)
		assert.Equal(t, "anthropic", result.LLM.Provider)
		assert.Equal(t, "claude-3-5-sonnet", result.LLM.Model)
		assert.Equal(t, "output", result.LLM.TokenType)
	})
}

func TestConvertUnitCostRoundTrip(t *testing.T) {
	t.Run("manual round trip", func(t *testing.T) {
		original := &feature.UnitCost{
			Type: feature.UnitCostTypeManual,
			Manual: &feature.ManualUnitCost{
				Amount: alpacadecimal.NewFromFloat(1.50),
			},
		}

		apiUC, err := convertUnitCostToAPI(original)
		require.NoError(t, err)

		result, err := convertUnitCostFromAPI(&apiUC)
		require.NoError(t, err)

		assert.Equal(t, original.Type, result.Type)
		assert.True(t, original.Manual.Amount.Equal(result.Manual.Amount))
	})

	t.Run("llm round trip", func(t *testing.T) {
		original := &feature.UnitCost{
			Type: feature.UnitCostTypeLLM,
			LLM: &feature.LLMUnitCost{
				ProviderProperty:  "provider",
				ModelProperty:     "model",
				TokenTypeProperty: "type",
			},
		}

		apiUC, err := convertUnitCostToAPI(original)
		require.NoError(t, err)

		result, err := convertUnitCostFromAPI(&apiUC)
		require.NoError(t, err)

		assert.Equal(t, original.Type, result.Type)
		assert.Equal(t, original.LLM, result.LLM)
	})
}

func TestConvertFeatureToAPI(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)

	t.Run("feature without meter or unit cost", func(t *testing.T) {
		f := feature.Feature{
			Namespace: "default",
			ID:        "feat-1",
			Name:      "My Feature",
			Key:       "my_feature",
			Metadata:  map[string]string{"env": "test"},
			CreatedAt: now,
			UpdatedAt: now,
		}

		result, err := convertFeatureToAPI(f)
		require.NoError(t, err)
		assert.Equal(t, "feat-1", result.Id)
		assert.Equal(t, api.ResourceKey("my_feature"), result.Key)
		assert.Equal(t, "My Feature", result.Name)
		assert.Nil(t, result.Meter)
		assert.Nil(t, result.UnitCost)
		assert.Nil(t, result.DeletedAt)
		require.NotNil(t, result.Labels)
		assert.Equal(t, "test", (*result.Labels)["env"])
	})

	t.Run("feature with meter and filters", func(t *testing.T) {
		f := feature.Feature{
			Namespace: "default",
			ID:        "feat-2",
			Name:      "Token Feature",
			Key:       "tokens",
			MeterID:   lo.ToPtr("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
			MeterGroupByFilters: feature.MeterGroupByFilters{
				"provider": {Eq: lo.ToPtr("openai")},
			},
			CreatedAt: now,
			UpdatedAt: now,
		}

		result, err := convertFeatureToAPI(f)
		require.NoError(t, err)
		require.NotNil(t, result.Meter)
		assert.Equal(t, api.ULID("01ARZ3NDEKTSV4RRFFQ69G5FAV"), result.Meter.Id)
		require.NotNil(t, result.Meter.Filters)
		filterMap := *result.Meter.Filters
		assert.Equal(t, lo.ToPtr("openai"), filterMap["provider"].Eq)
	})

	t.Run("feature with manual unit cost", func(t *testing.T) {
		f := feature.Feature{
			Namespace: "default",
			ID:        "feat-3",
			Name:      "API Calls",
			Key:       "api_calls",
			MeterID:   lo.ToPtr("api_requests"),
			UnitCost: &feature.UnitCost{
				Type: feature.UnitCostTypeManual,
				Manual: &feature.ManualUnitCost{
					Amount: alpacadecimal.NewFromFloat(0.01),
				},
			},
			CreatedAt: now,
			UpdatedAt: now,
		}

		result, err := convertFeatureToAPI(f)
		require.NoError(t, err)
		require.NotNil(t, result.UnitCost)

		disc, err := result.UnitCost.Discriminator()
		require.NoError(t, err)
		assert.Equal(t, "manual", disc)
	})

	t.Run("feature with archived at", func(t *testing.T) {
		archived := now.Add(-time.Hour)
		f := feature.Feature{
			Namespace:  "default",
			ID:         "feat-4",
			Name:       "Archived",
			Key:        "archived",
			ArchivedAt: &archived,
			CreatedAt:  now,
			UpdatedAt:  now,
		}

		result, err := convertFeatureToAPI(f)
		require.NoError(t, err)
		require.NotNil(t, result.DeletedAt)
		assert.Equal(t, archived, *result.DeletedAt)
	})
}

func TestConvertCreateRequestToDomain(t *testing.T) {
	t.Run("minimal request without meter", func(t *testing.T) {
		body := api.CreateFeatureRequest{
			Key:  "my_key",
			Name: "My Feature",
		}

		result, err := convertCreateRequestToDomain("test-ns", body, nil)
		require.NoError(t, err)
		assert.Equal(t, "test-ns", result.Namespace)
		assert.Equal(t, "my_key", result.Key)
		assert.Equal(t, "My Feature", result.Name)
		assert.Nil(t, result.MeterID)
		assert.Nil(t, result.UnitCost)
	})

	t.Run("with meter ID and filters", func(t *testing.T) {
		meterID := "01ARZ3NDEKTSV4RRFFQ69G5FAV"
		body := api.CreateFeatureRequest{
			Key:  "tokens",
			Name: "Tokens",
			Meter: &struct {
				Filters *map[string]api.QueryFilterStringMapItem `json:"filters,omitempty"`
				Id      api.ULID                                 `json:"id"`
			}{
				Id: api.ULID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
				Filters: &map[string]api.QueryFilterStringMapItem{
					"model": {Eq: lo.ToPtr("gpt-4")},
				},
			},
		}

		result, err := convertCreateRequestToDomain("ns", body, &meterID)
		require.NoError(t, err)
		require.NotNil(t, result.MeterID)
		assert.Equal(t, "01ARZ3NDEKTSV4RRFFQ69G5FAV", *result.MeterID)
		require.NotNil(t, result.MeterGroupByFilters)
		assert.Equal(t, lo.ToPtr("gpt-4"), result.MeterGroupByFilters["model"].Eq)
	})

	t.Run("with manual unit cost", func(t *testing.T) {
		var uc api.BillingFeatureUnitCost
		err := uc.FromBillingFeatureManualUnitCost(api.BillingFeatureManualUnitCost{Amount: "0.05"})
		require.NoError(t, err)

		body := api.CreateFeatureRequest{
			Key:      "feat",
			Name:     "Feature",
			UnitCost: &uc,
		}

		result, err := convertCreateRequestToDomain("ns", body, nil)
		require.NoError(t, err)
		require.NotNil(t, result.UnitCost)
		assert.Equal(t, feature.UnitCostTypeManual, result.UnitCost.Type)
		assert.Equal(t, "0.05", result.UnitCost.Manual.Amount.String())
	})

	t.Run("with labels", func(t *testing.T) {
		labels := api.Labels{"env": "prod", "team": "billing"}
		body := api.CreateFeatureRequest{
			Key:    "feat",
			Name:   "Feature",
			Labels: &labels,
		}

		result, err := convertCreateRequestToDomain("ns", body, nil)
		require.NoError(t, err)
		assert.Equal(t, "prod", result.Metadata["env"])
		assert.Equal(t, "billing", result.Metadata["team"])
	})
}

func TestEnrichFeatureResponseWithPricing(t *testing.T) {
	t.Run("adds pricing to llm unit cost", func(t *testing.T) {
		var uc api.BillingFeatureUnitCost
		err := uc.FromBillingFeatureLLMUnitCost(api.BillingFeatureLLMUnitCost{
			ProviderProperty: lo.ToPtr("provider"),
			ModelProperty:    lo.ToPtr("model"),
		})
		require.NoError(t, err)

		resp := &api.Feature{UnitCost: &uc}
		pricing := &llmcost.ModelPricing{
			InputPerToken:  alpacadecimal.NewFromFloat(0.00001),
			OutputPerToken: alpacadecimal.NewFromFloat(0.00003),
		}

		enrichFeatureResponseWithPricing(resp, pricing)

		llm, err := resp.UnitCost.AsBillingFeatureLLMUnitCost()
		require.NoError(t, err)
		require.NotNil(t, llm.Pricing)
		assert.Equal(t, api.Numeric("0.00001"), llm.Pricing.InputPerToken)
		assert.Equal(t, api.Numeric("0.00003"), llm.Pricing.OutputPerToken)
		assert.Nil(t, llm.Pricing.CacheReadPerToken)
	})

	t.Run("adds optional pricing fields", func(t *testing.T) {
		var uc api.BillingFeatureUnitCost
		err := uc.FromBillingFeatureLLMUnitCost(api.BillingFeatureLLMUnitCost{
			Provider: lo.ToPtr("openai"),
			Model:    lo.ToPtr("gpt-4"),
		})
		require.NoError(t, err)

		resp := &api.Feature{UnitCost: &uc}
		pricing := &llmcost.ModelPricing{
			InputPerToken:     alpacadecimal.NewFromFloat(0.00001),
			OutputPerToken:    alpacadecimal.NewFromFloat(0.00003),
			CacheReadPerToken: lo.ToPtr(alpacadecimal.NewFromFloat(0.000005)),
			ReasoningPerToken: lo.ToPtr(alpacadecimal.NewFromFloat(0.00006)),
		}

		enrichFeatureResponseWithPricing(resp, pricing)

		llm, err := resp.UnitCost.AsBillingFeatureLLMUnitCost()
		require.NoError(t, err)
		require.NotNil(t, llm.Pricing.CacheReadPerToken)
		assert.Equal(t, api.Numeric("0.000005"), *llm.Pricing.CacheReadPerToken)
		require.NotNil(t, llm.Pricing.ReasoningPerToken)
		assert.Equal(t, api.Numeric("0.00006"), *llm.Pricing.ReasoningPerToken)
		assert.Nil(t, llm.Pricing.CacheWritePerToken)
	})

	t.Run("no-op when unit cost is manual", func(t *testing.T) {
		var uc api.BillingFeatureUnitCost
		err := uc.FromBillingFeatureManualUnitCost(api.BillingFeatureManualUnitCost{Amount: "0.005"})
		require.NoError(t, err)
		resp := &api.Feature{UnitCost: &uc}

		enrichFeatureResponseWithPricing(resp, &llmcost.ModelPricing{
			InputPerToken:  alpacadecimal.NewFromFloat(0.00001),
			OutputPerToken: alpacadecimal.NewFromFloat(0.00003),
		})

		// Manual cost should be unchanged
		manual, err := resp.UnitCost.AsBillingFeatureManualUnitCost()
		require.NoError(t, err)
		assert.Equal(t, api.Numeric("0.005"), manual.Amount)
	})

	t.Run("no-op when unit cost is nil", func(t *testing.T) {
		resp := &api.Feature{}
		enrichFeatureResponseWithPricing(resp, &llmcost.ModelPricing{})
		assert.Nil(t, resp.UnitCost)
	})

	t.Run("no-op when pricing is nil", func(t *testing.T) {
		var uc api.BillingFeatureUnitCost
		err := uc.FromBillingFeatureLLMUnitCost(api.BillingFeatureLLMUnitCost{})
		require.NoError(t, err)
		resp := &api.Feature{UnitCost: &uc}
		enrichFeatureResponseWithPricing(resp, nil)

		llm, err := resp.UnitCost.AsBillingFeatureLLMUnitCost()
		require.NoError(t, err)
		assert.Nil(t, llm.Pricing)
	})
}

func TestExtractEqFilterValue(t *testing.T) {
	t.Run("returns value for eq filter", func(t *testing.T) {
		filters := feature.MeterGroupByFilters{
			"provider": {Eq: lo.ToPtr("openai")},
		}
		assert.Equal(t, "openai", extractEqFilterValue(filters, "provider"))
	})

	t.Run("returns empty for missing key", func(t *testing.T) {
		filters := feature.MeterGroupByFilters{
			"provider": {Eq: lo.ToPtr("openai")},
		}
		assert.Equal(t, "", extractEqFilterValue(filters, "model"))
	})

	t.Run("returns empty for non-eq filter", func(t *testing.T) {
		filters := feature.MeterGroupByFilters{
			"provider": {In: &[]string{"openai", "anthropic"}},
		}
		assert.Equal(t, "", extractEqFilterValue(filters, "provider"))
	})

	t.Run("returns empty for nil filters", func(t *testing.T) {
		assert.Equal(t, "", extractEqFilterValue(nil, "key"))
	})
}

func TestConvertFiltersRoundTrip(t *testing.T) {
	original := feature.MeterGroupByFilters{
		"provider": {Eq: lo.ToPtr("openai")},
		"model":    {In: &[]string{"gpt-4", "gpt-4o"}},
	}

	apiFilters := convertFiltersToAPI(original)
	assert.Len(t, apiFilters, 2)
	assert.Equal(t, lo.ToPtr("openai"), apiFilters["provider"].Eq)
	assert.Equal(t, &[]string{"gpt-4", "gpt-4o"}, apiFilters["model"].In)

	roundTripped := convertFiltersFromAPI(apiFilters)
	assert.Equal(t, lo.ToPtr("openai"), roundTripped["provider"].Eq)
	assert.Equal(t, &[]string{"gpt-4", "gpt-4o"}, roundTripped["model"].In)
}

func TestConvertMetadataLabels(t *testing.T) {
	t.Run("metadata to labels", func(t *testing.T) {
		labels := convertMetadataToLabels(map[string]string{"a": "1", "b": "2"})
		require.NotNil(t, labels)
		assert.Equal(t, "1", (*labels)["a"])
		assert.Equal(t, "2", (*labels)["b"])
	})

	t.Run("nil metadata returns nil labels", func(t *testing.T) {
		labels := convertMetadataToLabels(nil)
		assert.Nil(t, labels)
	})

	t.Run("labels to metadata", func(t *testing.T) {
		labels := api.Labels{"x": "y"}
		meta := convertLabelsToMetadata(&labels)
		assert.Equal(t, "y", meta["x"])
	})

	t.Run("nil labels returns nil metadata", func(t *testing.T) {
		meta := convertLabelsToMetadata(nil)
		assert.Nil(t, meta)
	})
}

func TestConvertFilterStringToAPIMapItem(t *testing.T) {
	t.Run("eq filter", func(t *testing.T) {
		f := filter.FilterString{Eq: lo.ToPtr("val")}
		result := convertFilterStringToAPIMapItem(f)
		assert.Equal(t, lo.ToPtr("val"), result.Eq)
	})

	t.Run("in filter", func(t *testing.T) {
		f := filter.FilterString{In: &[]string{"a", "b"}}
		result := convertFilterStringToAPIMapItem(f)
		assert.Equal(t, &[]string{"a", "b"}, result.In)
	})

	t.Run("ne filter", func(t *testing.T) {
		f := filter.FilterString{Ne: lo.ToPtr("excluded")}
		result := convertFilterStringToAPIMapItem(f)
		assert.Equal(t, lo.ToPtr("excluded"), result.Neq)
	})

	t.Run("exists filter", func(t *testing.T) {
		f := filter.FilterString{Exists: lo.ToPtr(true)}
		result := convertFilterStringToAPIMapItem(f)
		assert.Equal(t, lo.ToPtr(true), result.Exists)
	})
}
