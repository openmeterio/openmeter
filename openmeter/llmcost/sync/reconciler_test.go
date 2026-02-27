package sync

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/llmcost"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// mockAdapter implements llmcost.Adapter for testing.
type mockAdapter struct {
	upsertedPrices []llmcost.Price
	upsertErr      error
}

func (m *mockAdapter) Tx(_ context.Context) (context.Context, transaction.Driver, error) {
	return context.Background(), nil, nil
}

func (m *mockAdapter) UpsertGlobalPrice(_ context.Context, price llmcost.Price) error {
	if m.upsertErr != nil {
		return m.upsertErr
	}
	m.upsertedPrices = append(m.upsertedPrices, price)
	return nil
}

func (m *mockAdapter) ListPrices(context.Context, llmcost.ListPricesInput) (pagination.Result[llmcost.Price], error) {
	return pagination.Result[llmcost.Price]{}, nil
}

func (m *mockAdapter) GetPrice(context.Context, llmcost.GetPriceInput) (llmcost.Price, error) {
	return llmcost.Price{}, nil
}

func (m *mockAdapter) ResolvePrice(context.Context, llmcost.ResolvePriceInput) (llmcost.Price, error) {
	return llmcost.Price{}, nil
}

func (m *mockAdapter) CreateOverride(context.Context, llmcost.CreateOverrideInput) (llmcost.Price, error) {
	return llmcost.Price{}, nil
}

func (m *mockAdapter) UpdateOverride(context.Context, llmcost.UpdateOverrideInput) (llmcost.Price, error) {
	return llmcost.Price{}, nil
}

func (m *mockAdapter) DeleteOverride(context.Context, llmcost.DeleteOverrideInput) error {
	return nil
}

func (m *mockAdapter) ListOverrides(context.Context, llmcost.ListOverridesInput) (pagination.Result[llmcost.Price], error) {
	return pagination.Result[llmcost.Price]{}, nil
}

func makePrice(source string, provider string, modelID string, input, output float64) llmcost.SourcePrice {
	return llmcost.SourcePrice{
		Source:   llmcost.PriceSource(source),
		Provider: llmcost.Provider(provider),
		ModelID:  modelID,
		Pricing: llmcost.ModelPricing{
			InputPerToken:  alpacadecimal.NewFromFloat(input),
			OutputPerToken: alpacadecimal.NewFromFloat(output),
		},
		FetchedAt: time.Now(),
	}
}

func TestReconcileAgreement(t *testing.T) {
	adapter := &mockAdapter{}
	logger := slog.Default()

	t.Run("two sources agree within tolerance", func(t *testing.T) {
		adapter.upsertedPrices = nil
		r := NewReconciler(adapter, logger, 2, 0.01)

		prices := []llmcost.SourcePrice{
			makePrice("source_a", "openai", "gpt-4", 0.01, 0.03),
			makePrice("source_b", "openai", "gpt-4", 0.01, 0.03),
		}

		err := r.Reconcile(context.Background(), prices)
		require.NoError(t, err)
		require.Len(t, adapter.upsertedPrices, 1)

		upserted := adapter.upsertedPrices[0]
		assert.Equal(t, "openai", string(upserted.Provider))
		assert.Equal(t, "gpt-4", upserted.ModelID)
		assert.Equal(t, llmcost.PriceSourceSystem, upserted.Source)
		assert.True(t, upserted.Pricing.InputPerToken.Equal(alpacadecimal.NewFromFloat(0.01)))
		assert.True(t, upserted.Pricing.OutputPerToken.Equal(alpacadecimal.NewFromFloat(0.03)))
		assert.Len(t, upserted.SourcePrices, 2)
	})

	t.Run("two sources disagree beyond tolerance", func(t *testing.T) {
		adapter.upsertedPrices = nil
		r := NewReconciler(adapter, logger, 2, 0.01) // 1% tolerance

		prices := []llmcost.SourcePrice{
			makePrice("source_a", "openai", "gpt-4", 0.01, 0.03),
			makePrice("source_b", "openai", "gpt-4", 0.02, 0.03), // input 100% different
		}

		err := r.Reconcile(context.Background(), prices)
		require.NoError(t, err)
		assert.Empty(t, adapter.upsertedPrices)
	})

	t.Run("not enough sources", func(t *testing.T) {
		adapter.upsertedPrices = nil
		r := NewReconciler(adapter, logger, 2, 0.01)

		prices := []llmcost.SourcePrice{
			makePrice("source_a", "openai", "gpt-4", 0.01, 0.03),
		}

		err := r.Reconcile(context.Background(), prices)
		require.NoError(t, err)
		assert.Empty(t, adapter.upsertedPrices)
	})

	t.Run("three sources two agree", func(t *testing.T) {
		adapter.upsertedPrices = nil
		r := NewReconciler(adapter, logger, 2, 0.01)

		prices := []llmcost.SourcePrice{
			makePrice("source_a", "openai", "gpt-4", 0.01, 0.03),
			makePrice("source_b", "openai", "gpt-4", 0.99, 0.99), // outlier
			makePrice("source_c", "openai", "gpt-4", 0.01, 0.03),
		}

		err := r.Reconcile(context.Background(), prices)
		require.NoError(t, err)
		require.Len(t, adapter.upsertedPrices, 1)

		upserted := adapter.upsertedPrices[0]
		assert.Len(t, upserted.SourcePrices, 2)
		_, hasA := upserted.SourcePrices["source_a"]
		_, hasC := upserted.SourcePrices["source_c"]
		assert.True(t, hasA)
		assert.True(t, hasC)
	})

	t.Run("multiple models reconciled independently", func(t *testing.T) {
		adapter.upsertedPrices = nil
		r := NewReconciler(adapter, logger, 2, 0.01)

		prices := []llmcost.SourcePrice{
			makePrice("source_a", "openai", "gpt-4", 0.01, 0.03),
			makePrice("source_b", "openai", "gpt-4", 0.01, 0.03),
			makePrice("source_a", "anthropic", "claude-3-5-sonnet", 0.003, 0.015),
			makePrice("source_b", "anthropic", "claude-3-5-sonnet", 0.003, 0.015),
		}

		err := r.Reconcile(context.Background(), prices)
		require.NoError(t, err)
		assert.Len(t, adapter.upsertedPrices, 2)
	})

	t.Run("averages agreeing prices", func(t *testing.T) {
		adapter.upsertedPrices = nil
		r := NewReconciler(adapter, logger, 2, 0.05) // 5% tolerance

		prices := []llmcost.SourcePrice{
			makePrice("source_a", "openai", "gpt-4", 0.010, 0.030),
			makePrice("source_b", "openai", "gpt-4", 0.0104, 0.0304), // within 5%
		}

		err := r.Reconcile(context.Background(), prices)
		require.NoError(t, err)
		require.Len(t, adapter.upsertedPrices, 1)

		upserted := adapter.upsertedPrices[0]
		expectedInput := alpacadecimal.NewFromFloat(0.010).Add(alpacadecimal.NewFromFloat(0.0104)).Div(alpacadecimal.NewFromInt(2))
		assert.True(t, upserted.Pricing.InputPerToken.Equal(expectedInput))
	})
}

func TestReconcilerDefaults(t *testing.T) {
	adapter := &mockAdapter{}
	logger := slog.Default()

	t.Run("zero minAgreement uses default", func(t *testing.T) {
		r := NewReconciler(adapter, logger, 0, 0.01)
		assert.Equal(t, DefaultMinSourceAgreement, r.minAgreement)
	})

	t.Run("negative priceTolerance uses default", func(t *testing.T) {
		r := NewReconciler(adapter, logger, 2, -1)
		assert.Equal(t, DefaultPriceTolerance, r.priceTolerance)
	})

	t.Run("zero tolerance is valid", func(t *testing.T) {
		r := NewReconciler(adapter, logger, 2, 0)
		assert.Equal(t, float64(0), r.priceTolerance)
	})
}

func TestDecimalsAgree(t *testing.T) {
	tolerance := alpacadecimal.NewFromFloat(0.01) // 1%

	t.Run("both zero", func(t *testing.T) {
		assert.True(t, decimalsAgree(
			alpacadecimal.NewFromInt(0),
			alpacadecimal.NewFromInt(0),
			tolerance,
		))
	})

	t.Run("one zero one nonzero", func(t *testing.T) {
		assert.False(t, decimalsAgree(
			alpacadecimal.NewFromInt(0),
			alpacadecimal.NewFromFloat(0.01),
			tolerance,
		))
	})

	t.Run("exact match", func(t *testing.T) {
		assert.True(t, decimalsAgree(
			alpacadecimal.NewFromFloat(0.01),
			alpacadecimal.NewFromFloat(0.01),
			tolerance,
		))
	})

	t.Run("within tolerance", func(t *testing.T) {
		// 0.01 vs 0.01005 = 0.5% difference
		assert.True(t, decimalsAgree(
			alpacadecimal.NewFromFloat(0.01),
			alpacadecimal.NewFromFloat(0.01005),
			tolerance,
		))
	})

	t.Run("beyond tolerance", func(t *testing.T) {
		// 0.01 vs 0.0102 = 2% difference
		assert.False(t, decimalsAgree(
			alpacadecimal.NewFromFloat(0.01),
			alpacadecimal.NewFromFloat(0.0102),
			tolerance,
		))
	})
}

func TestPricesAgree(t *testing.T) {
	tolerance := alpacadecimal.NewFromFloat(0.01)

	t.Run("both dimensions agree", func(t *testing.T) {
		a := llmcost.ModelPricing{
			InputPerToken:  alpacadecimal.NewFromFloat(0.01),
			OutputPerToken: alpacadecimal.NewFromFloat(0.03),
		}
		b := llmcost.ModelPricing{
			InputPerToken:  alpacadecimal.NewFromFloat(0.01),
			OutputPerToken: alpacadecimal.NewFromFloat(0.03),
		}
		assert.True(t, pricesAgree(a, b, tolerance))
	})

	t.Run("input disagrees", func(t *testing.T) {
		a := llmcost.ModelPricing{
			InputPerToken:  alpacadecimal.NewFromFloat(0.01),
			OutputPerToken: alpacadecimal.NewFromFloat(0.03),
		}
		b := llmcost.ModelPricing{
			InputPerToken:  alpacadecimal.NewFromFloat(0.02),
			OutputPerToken: alpacadecimal.NewFromFloat(0.03),
		}
		assert.False(t, pricesAgree(a, b, tolerance))
	})

	t.Run("output disagrees", func(t *testing.T) {
		a := llmcost.ModelPricing{
			InputPerToken:  alpacadecimal.NewFromFloat(0.01),
			OutputPerToken: alpacadecimal.NewFromFloat(0.03),
		}
		b := llmcost.ModelPricing{
			InputPerToken:  alpacadecimal.NewFromFloat(0.01),
			OutputPerToken: alpacadecimal.NewFromFloat(0.06),
		}
		assert.False(t, pricesAgree(a, b, tolerance))
	})
}

func TestAveragePrices(t *testing.T) {
	t.Run("single price returns itself", func(t *testing.T) {
		prices := []llmcost.SourcePrice{
			makePrice("source_a", "openai", "gpt-4", 0.01, 0.03),
		}

		avg, err := averagePrices(prices)
		require.NoError(t, err)
		assert.True(t, avg.Pricing.InputPerToken.Equal(alpacadecimal.NewFromFloat(0.01)))
		assert.True(t, avg.Pricing.OutputPerToken.Equal(alpacadecimal.NewFromFloat(0.03)))
	})

	t.Run("averages two prices", func(t *testing.T) {
		prices := []llmcost.SourcePrice{
			makePrice("source_a", "openai", "gpt-4", 0.01, 0.03),
			makePrice("source_b", "openai", "gpt-4", 0.02, 0.04),
		}

		avg, err := averagePrices(prices)
		require.NoError(t, err)
		assert.True(t, avg.Pricing.InputPerToken.Equal(alpacadecimal.NewFromFloat(0.015)))
		assert.True(t, avg.Pricing.OutputPerToken.Equal(alpacadecimal.NewFromFloat(0.035)))
	})

	t.Run("uses first price metadata", func(t *testing.T) {
		prices := []llmcost.SourcePrice{
			{
				Source:    "source_a",
				Provider:  "openai",
				ModelID:   "gpt-4",
				ModelName: "GPT-4",
				Pricing: llmcost.ModelPricing{
					InputPerToken:  alpacadecimal.NewFromFloat(0.01),
					OutputPerToken: alpacadecimal.NewFromFloat(0.03),
				},
			},
			{
				Source:    "source_b",
				Provider:  "openai",
				ModelID:   "gpt-4",
				ModelName: "GPT 4",
				Pricing: llmcost.ModelPricing{
					InputPerToken:  alpacadecimal.NewFromFloat(0.01),
					OutputPerToken: alpacadecimal.NewFromFloat(0.03),
				},
			},
		}

		avg, err := averagePrices(prices)
		require.NoError(t, err)
		assert.Equal(t, "GPT-4", avg.ModelName)
	})
}
