package sync

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
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

func makePriceWithOptional(source, provider, modelID string, input, output float64, cacheRead, cacheWrite, reasoning *float64) llmcost.SourcePrice {
	p := makePrice(source, provider, modelID, input, output)
	if cacheRead != nil {
		p.Pricing.CacheReadPerToken = lo.ToPtr(alpacadecimal.NewFromFloat(*cacheRead))
	}
	if cacheWrite != nil {
		p.Pricing.CacheWritePerToken = lo.ToPtr(alpacadecimal.NewFromFloat(*cacheWrite))
	}
	if reasoning != nil {
		p.Pricing.ReasoningPerToken = lo.ToPtr(alpacadecimal.NewFromFloat(*reasoning))
	}
	return p
}

func fp(v float64) *float64 { return &v }

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

	t.Run("optional fields both nil agree", func(t *testing.T) {
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

	t.Run("optional fields both set and agree", func(t *testing.T) {
		cacheRead := alpacadecimal.NewFromFloat(0.005)
		reasoning := alpacadecimal.NewFromFloat(0.02)
		a := llmcost.ModelPricing{
			InputPerToken:     alpacadecimal.NewFromFloat(0.01),
			OutputPerToken:    alpacadecimal.NewFromFloat(0.03),
			CacheReadPerToken: &cacheRead,
			ReasoningPerToken: &reasoning,
		}
		b := llmcost.ModelPricing{
			InputPerToken:     alpacadecimal.NewFromFloat(0.01),
			OutputPerToken:    alpacadecimal.NewFromFloat(0.03),
			CacheReadPerToken: &cacheRead,
			ReasoningPerToken: &reasoning,
		}
		assert.True(t, pricesAgree(a, b, tolerance))
	})

	t.Run("optional field one nil one set disagrees", func(t *testing.T) {
		cacheRead := alpacadecimal.NewFromFloat(0.005)
		a := llmcost.ModelPricing{
			InputPerToken:     alpacadecimal.NewFromFloat(0.01),
			OutputPerToken:    alpacadecimal.NewFromFloat(0.03),
			CacheReadPerToken: &cacheRead,
		}
		b := llmcost.ModelPricing{
			InputPerToken:  alpacadecimal.NewFromFloat(0.01),
			OutputPerToken: alpacadecimal.NewFromFloat(0.03),
		}
		assert.False(t, pricesAgree(a, b, tolerance))
	})

	t.Run("optional fields both set but disagree", func(t *testing.T) {
		cacheA := alpacadecimal.NewFromFloat(0.005)
		cacheB := alpacadecimal.NewFromFloat(0.010) // 100% off
		a := llmcost.ModelPricing{
			InputPerToken:     alpacadecimal.NewFromFloat(0.01),
			OutputPerToken:    alpacadecimal.NewFromFloat(0.03),
			CacheReadPerToken: &cacheA,
		}
		b := llmcost.ModelPricing{
			InputPerToken:     alpacadecimal.NewFromFloat(0.01),
			OutputPerToken:    alpacadecimal.NewFromFloat(0.03),
			CacheReadPerToken: &cacheB,
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

	t.Run("averages optional fields", func(t *testing.T) {
		prices := []llmcost.SourcePrice{
			makePriceWithOptional("source_a", "openai", "gpt-4", 0.01, 0.03, fp(0.004), fp(0.006), fp(0.02)),
			makePriceWithOptional("source_b", "openai", "gpt-4", 0.02, 0.04, fp(0.006), fp(0.008), fp(0.04)),
		}

		avg, err := averagePrices(prices)
		require.NoError(t, err)
		require.NotNil(t, avg.Pricing.CacheReadPerToken)
		require.NotNil(t, avg.Pricing.CacheWritePerToken)
		require.NotNil(t, avg.Pricing.ReasoningPerToken)
		assert.True(t, avg.Pricing.CacheReadPerToken.Equal(alpacadecimal.NewFromFloat(0.005)))
		assert.True(t, avg.Pricing.CacheWritePerToken.Equal(alpacadecimal.NewFromFloat(0.007)))
		assert.True(t, avg.Pricing.ReasoningPerToken.Equal(alpacadecimal.NewFromFloat(0.03)))
	})

	t.Run("nil optional fields stay nil", func(t *testing.T) {
		prices := []llmcost.SourcePrice{
			makePrice("source_a", "openai", "gpt-4", 0.01, 0.03),
			makePrice("source_b", "openai", "gpt-4", 0.02, 0.04),
		}

		avg, err := averagePrices(prices)
		require.NoError(t, err)
		assert.Nil(t, avg.Pricing.CacheReadPerToken)
		assert.Nil(t, avg.Pricing.CacheWritePerToken)
		assert.Nil(t, avg.Pricing.ReasoningPerToken)
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

func TestOptionalDecimalsAgree(t *testing.T) {
	tolerance := alpacadecimal.NewFromFloat(0.01)

	t.Run("both nil", func(t *testing.T) {
		assert.True(t, optionalDecimalsAgree(nil, nil, tolerance))
	})

	t.Run("first nil second set", func(t *testing.T) {
		v := alpacadecimal.NewFromFloat(0.01)
		assert.False(t, optionalDecimalsAgree(nil, &v, tolerance))
	})

	t.Run("first set second nil", func(t *testing.T) {
		v := alpacadecimal.NewFromFloat(0.01)
		assert.False(t, optionalDecimalsAgree(&v, nil, tolerance))
	})

	t.Run("both set and agree", func(t *testing.T) {
		a := alpacadecimal.NewFromFloat(0.01)
		b := alpacadecimal.NewFromFloat(0.01)
		assert.True(t, optionalDecimalsAgree(&a, &b, tolerance))
	})

	t.Run("both set and disagree", func(t *testing.T) {
		a := alpacadecimal.NewFromFloat(0.01)
		b := alpacadecimal.NewFromFloat(0.02)
		assert.False(t, optionalDecimalsAgree(&a, &b, tolerance))
	})

	t.Run("both set zero", func(t *testing.T) {
		a := alpacadecimal.NewFromInt(0)
		b := alpacadecimal.NewFromInt(0)
		assert.True(t, optionalDecimalsAgree(&a, &b, tolerance))
	})
}

func TestReconcileUpsertError(t *testing.T) {
	logger := slog.Default()

	t.Run("returns error on upsert failure", func(t *testing.T) {
		adapter := &mockAdapter{upsertErr: errors.New("db connection lost")}
		r := NewReconciler(adapter, logger, 2, 0.01)

		prices := []llmcost.SourcePrice{
			makePrice("source_a", "openai", "gpt-4", 0.01, 0.03),
			makePrice("source_b", "openai", "gpt-4", 0.01, 0.03),
		}

		err := r.Reconcile(context.Background(), prices)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db connection lost")
		assert.Contains(t, err.Error(), "openai/gpt-4")
	})

	t.Run("continues reconciling other models after upsert failure", func(t *testing.T) {
		// Use a mock that fails on first call then succeeds
		callCount := 0
		adapter := &mockAdapter{}
		origUpsert := adapter.UpsertGlobalPrice
		_ = origUpsert

		// Custom adapter that fails once
		failOnceAdapter := &failOnceMockAdapter{}
		r := NewReconciler(failOnceAdapter, logger, 2, 0.01)

		prices := []llmcost.SourcePrice{
			makePrice("source_a", "openai", "gpt-4", 0.01, 0.03),
			makePrice("source_b", "openai", "gpt-4", 0.01, 0.03),
			makePrice("source_a", "anthropic", "claude-3", 0.003, 0.015),
			makePrice("source_b", "anthropic", "claude-3", 0.003, 0.015),
		}

		err := r.Reconcile(context.Background(), prices)
		// Should return error for the failed upsert
		require.Error(t, err)
		// But should still have attempted the successful one
		assert.Len(t, failOnceAdapter.upsertedPrices, 1)
		_ = callCount
	})
}

func TestReconcileWithOptionalFields(t *testing.T) {
	logger := slog.Default()

	t.Run("reconciles prices with optional fields", func(t *testing.T) {
		adapter := &mockAdapter{}
		r := NewReconciler(adapter, logger, 2, 0.01)

		prices := []llmcost.SourcePrice{
			makePriceWithOptional("source_a", "openai", "gpt-4", 0.01, 0.03, fp(0.005), nil, fp(0.02)),
			makePriceWithOptional("source_b", "openai", "gpt-4", 0.01, 0.03, fp(0.005), nil, fp(0.02)),
		}

		err := r.Reconcile(context.Background(), prices)
		require.NoError(t, err)
		require.Len(t, adapter.upsertedPrices, 1)

		upserted := adapter.upsertedPrices[0]
		require.NotNil(t, upserted.Pricing.CacheReadPerToken)
		assert.True(t, upserted.Pricing.CacheReadPerToken.Equal(alpacadecimal.NewFromFloat(0.005)))
		assert.Nil(t, upserted.Pricing.CacheWritePerToken)
		require.NotNil(t, upserted.Pricing.ReasoningPerToken)
		assert.True(t, upserted.Pricing.ReasoningPerToken.Equal(alpacadecimal.NewFromFloat(0.02)))
	})

	t.Run("optional field disagreement prevents reconciliation", func(t *testing.T) {
		adapter := &mockAdapter{}
		r := NewReconciler(adapter, logger, 2, 0.01)

		// Sources agree on input/output but disagree on cache_read
		prices := []llmcost.SourcePrice{
			makePriceWithOptional("source_a", "openai", "gpt-4", 0.01, 0.03, fp(0.005), nil, nil),
			makePriceWithOptional("source_b", "openai", "gpt-4", 0.01, 0.03, fp(0.010), nil, nil), // 100% diff on cache_read
		}

		err := r.Reconcile(context.Background(), prices)
		require.NoError(t, err)
		assert.Empty(t, adapter.upsertedPrices)
	})

	t.Run("one source has optional field other does not prevents reconciliation", func(t *testing.T) {
		adapter := &mockAdapter{}
		r := NewReconciler(adapter, logger, 2, 0.01)

		prices := []llmcost.SourcePrice{
			makePriceWithOptional("source_a", "openai", "gpt-4", 0.01, 0.03, fp(0.005), nil, nil),
			makePrice("source_b", "openai", "gpt-4", 0.01, 0.03), // no cache_read
		}

		err := r.Reconcile(context.Background(), prices)
		require.NoError(t, err)
		assert.Empty(t, adapter.upsertedPrices)
	})
}

// failOnceMockAdapter fails the first UpsertGlobalPrice call, succeeds after.
type failOnceMockAdapter struct {
	mockAdapter
	called int
}

func (m *failOnceMockAdapter) UpsertGlobalPrice(_ context.Context, price llmcost.Price) error {
	m.called++
	if m.called == 1 {
		return errors.New("transient db error")
	}
	m.upsertedPrices = append(m.upsertedPrices, price)
	return nil
}
