package llmcost

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPriceSourceValidate(t *testing.T) {
	t.Run("valid sources", func(t *testing.T) {
		assert.NoError(t, PriceSourceManual.Validate())
		assert.NoError(t, PriceSourceSystem.Validate())
		assert.NoError(t, PriceSource("custom_fetcher").Validate())
	})

	t.Run("empty source", func(t *testing.T) {
		err := PriceSource("").Validate()
		require.Error(t, err)
	})
}

func TestModelPricingValidate(t *testing.T) {
	t.Run("valid pricing", func(t *testing.T) {
		p := ModelPricing{
			InputPerToken:  alpacadecimal.NewFromFloat(0.001),
			OutputPerToken: alpacadecimal.NewFromFloat(0.002),
		}
		assert.NoError(t, p.Validate())
	})

	t.Run("zero pricing is valid", func(t *testing.T) {
		p := ModelPricing{}
		assert.NoError(t, p.Validate())
	})

	t.Run("negative input price", func(t *testing.T) {
		p := ModelPricing{
			InputPerToken:  alpacadecimal.NewFromFloat(-0.001),
			OutputPerToken: alpacadecimal.NewFromFloat(0.002),
		}
		require.Error(t, p.Validate())
	})

	t.Run("negative output price", func(t *testing.T) {
		p := ModelPricing{
			InputPerToken:  alpacadecimal.NewFromFloat(0.001),
			OutputPerToken: alpacadecimal.NewFromFloat(-0.002),
		}
		require.Error(t, p.Validate())
	})
}

func TestPriceValidate(t *testing.T) {
	validPricing := ModelPricing{
		InputPerToken:  alpacadecimal.NewFromFloat(0.001),
		OutputPerToken: alpacadecimal.NewFromFloat(0.002),
	}

	t.Run("valid price", func(t *testing.T) {
		p := Price{
			Provider:      "openai",
			ModelID:       "gpt-4",
			Pricing:       validPricing,
			Source:        PriceSourceSystem,
			EffectiveFrom: time.Now(),
		}
		assert.NoError(t, p.Validate())
	})

	t.Run("empty provider", func(t *testing.T) {
		p := Price{
			ModelID:       "gpt-4",
			Pricing:       validPricing,
			Source:        PriceSourceSystem,
			EffectiveFrom: time.Now(),
		}
		require.Error(t, p.Validate())
	})

	t.Run("empty model ID", func(t *testing.T) {
		p := Price{
			Provider:      "openai",
			Pricing:       validPricing,
			Source:        PriceSourceSystem,
			EffectiveFrom: time.Now(),
		}
		require.Error(t, p.Validate())
	})

	t.Run("empty source", func(t *testing.T) {
		p := Price{
			Provider:      "openai",
			ModelID:       "gpt-4",
			Pricing:       validPricing,
			EffectiveFrom: time.Now(),
		}
		require.Error(t, p.Validate())
	})

	t.Run("effective_from after effective_to", func(t *testing.T) {
		now := time.Now()
		past := now.Add(-time.Hour)
		p := Price{
			Provider:      "openai",
			ModelID:       "gpt-4",
			Pricing:       validPricing,
			Source:        PriceSourceSystem,
			EffectiveFrom: now,
			EffectiveTo:   &past,
		}
		require.Error(t, p.Validate())
	})

	t.Run("effective_from before effective_to is valid", func(t *testing.T) {
		now := time.Now()
		future := now.Add(time.Hour)
		p := Price{
			Provider:      "openai",
			ModelID:       "gpt-4",
			Pricing:       validPricing,
			Source:        PriceSourceSystem,
			EffectiveFrom: now,
			EffectiveTo:   &future,
		}
		assert.NoError(t, p.Validate())
	})
}

func TestInputValidation(t *testing.T) {
	t.Run("ListPricesInput", func(t *testing.T) {
		// Empty input is valid (no required fields)
		assert.NoError(t, ListPricesInput{}.Validate())
	})

	t.Run("GetPriceInput", func(t *testing.T) {
		assert.NoError(t, GetPriceInput{ID: "abc"}.Validate())
		require.Error(t, GetPriceInput{}.Validate())
	})

	t.Run("ResolvePriceInput", func(t *testing.T) {
		assert.NoError(t, ResolvePriceInput{
			Namespace: "ns",
			Provider:  "openai",
			ModelID:   "gpt-4",
		}.Validate())

		require.Error(t, ResolvePriceInput{}.Validate())
		require.Error(t, ResolvePriceInput{Namespace: "ns"}.Validate())
		require.Error(t, ResolvePriceInput{Namespace: "ns", Provider: "openai"}.Validate())
	})

	t.Run("CreateOverrideInput", func(t *testing.T) {
		assert.NoError(t, CreateOverrideInput{
			Namespace:     "ns",
			Provider:      "openai",
			ModelID:       "gpt-4",
			Pricing:       ModelPricing{InputPerToken: alpacadecimal.NewFromFloat(0.001), OutputPerToken: alpacadecimal.NewFromFloat(0.002)},
			EffectiveFrom: time.Now(),
		}.Validate())

		require.Error(t, CreateOverrideInput{}.Validate())
	})

	t.Run("CreateOverrideInput effective_from after effective_to", func(t *testing.T) {
		now := time.Now()
		past := now.Add(-time.Hour)
		require.Error(t, CreateOverrideInput{
			Namespace:     "ns",
			Provider:      "openai",
			ModelID:       "gpt-4",
			Pricing:       ModelPricing{InputPerToken: alpacadecimal.NewFromFloat(0.001), OutputPerToken: alpacadecimal.NewFromFloat(0.002)},
			EffectiveFrom: now,
			EffectiveTo:   &past,
		}.Validate())
	})

	t.Run("UpdateOverrideInput", func(t *testing.T) {
		assert.NoError(t, UpdateOverrideInput{
			ID:        "abc",
			Namespace: "ns",
			Pricing:   ModelPricing{InputPerToken: alpacadecimal.NewFromFloat(0.001), OutputPerToken: alpacadecimal.NewFromFloat(0.002)},
		}.Validate())

		require.Error(t, UpdateOverrideInput{}.Validate())
	})

	t.Run("DeleteOverrideInput", func(t *testing.T) {
		assert.NoError(t, DeleteOverrideInput{ID: "abc", Namespace: "ns"}.Validate())
		require.Error(t, DeleteOverrideInput{}.Validate())
		require.Error(t, DeleteOverrideInput{ID: "abc"}.Validate())
	})

	t.Run("ListOverridesInput", func(t *testing.T) {
		assert.NoError(t, ListOverridesInput{Namespace: "ns"}.Validate())
		require.Error(t, ListOverridesInput{}.Validate())
	})
}
