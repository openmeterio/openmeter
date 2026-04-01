package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api/v3/filters"
	"github.com/openmeterio/openmeter/openmeter/llmcost"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// noopDriver implements transaction.Driver as a no-op for unit tests.
type noopDriver struct{}

func (noopDriver) Commit() error    { return nil }
func (noopDriver) Rollback() error  { return nil }
func (noopDriver) SavePoint() error { return nil }

// mockAdapter implements llmcost.Adapter for testing the service layer.
type mockAdapter struct {
	prices    []llmcost.Price
	overrides []llmcost.Price
}

func (m *mockAdapter) Tx(_ context.Context) (context.Context, transaction.Driver, error) {
	return context.Background(), noopDriver{}, nil
}

func (m *mockAdapter) ListPrices(_ context.Context, _ llmcost.ListPricesInput) (pagination.Result[llmcost.Price], error) {
	return pagination.Result[llmcost.Price]{
		Items:      m.prices,
		TotalCount: len(m.prices),
	}, nil
}

func (m *mockAdapter) GetPrice(_ context.Context, _ llmcost.GetPriceInput) (llmcost.Price, error) {
	return llmcost.Price{}, nil
}

func (m *mockAdapter) ResolvePrice(_ context.Context, _ llmcost.ResolvePriceInput) (llmcost.Price, error) {
	return llmcost.Price{}, nil
}

func (m *mockAdapter) CreateOverride(_ context.Context, _ llmcost.CreateOverrideInput) (llmcost.Price, error) {
	return llmcost.Price{}, nil
}

func (m *mockAdapter) DeleteOverride(_ context.Context, _ llmcost.DeleteOverrideInput) error {
	return nil
}

func (m *mockAdapter) ListOverrides(_ context.Context, _ llmcost.ListOverridesInput) (pagination.Result[llmcost.Price], error) {
	return pagination.Result[llmcost.Price]{
		Items:      m.overrides,
		TotalCount: len(m.overrides),
	}, nil
}

func (m *mockAdapter) UpsertGlobalPrice(_ context.Context, _ llmcost.Price) error {
	return nil
}

func makeTestPrice(provider string, modelID string, source llmcost.PriceSource) llmcost.Price {
	return llmcost.Price{
		ID:        provider + "/" + modelID,
		Provider:  llmcost.Provider(provider),
		ModelID:   modelID,
		ModelName: modelID,
		Source:    source,
		Currency:  "USD",
		Pricing: llmcost.ModelPricing{
			InputPerToken:  alpacadecimal.NewFromFloat(0.001),
			OutputPerToken: alpacadecimal.NewFromFloat(0.002),
		},
		EffectiveFrom: time.Now(),
	}
}

func makeTestOverride(provider string, modelID string, ns string) llmcost.Price {
	p := makeTestPrice(provider, modelID, llmcost.PriceSourceManual)
	p.ID = "override-" + provider + "/" + modelID
	p.Namespace = &ns
	p.Pricing.InputPerToken = alpacadecimal.NewFromFloat(0.0005) // cheaper override
	p.Pricing.OutputPerToken = alpacadecimal.NewFromFloat(0.001)
	return p
}

func TestListPrices_SourceFilterOverlay(t *testing.T) {
	ns := "test-ns"

	systemPrice := makeTestPrice("openai", "gpt-4", llmcost.PriceSourceSystem)
	manualGlobalPrice := makeTestPrice("anthropic", "claude-3", llmcost.PriceSourceManual)
	override := makeTestOverride("openai", "gpt-4", ns)

	t.Run("no source filter applies overlay", func(t *testing.T) {
		adapter := &mockAdapter{
			prices:    []llmcost.Price{systemPrice},
			overrides: []llmcost.Price{override},
		}
		svc := New(adapter, slog.Default())

		result, err := svc.ListPrices(context.Background(), llmcost.ListPricesInput{
			Namespace: ns,
			Page:      pagination.NewPage(1, 20),
		})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		// Should be replaced by the override
		assert.Equal(t, override.ID, result.Items[0].ID)
		assert.Equal(t, llmcost.PriceSourceManual, result.Items[0].Source)
	})

	t.Run("source=system skips overlay", func(t *testing.T) {
		adapter := &mockAdapter{
			prices:    []llmcost.Price{systemPrice},
			overrides: []llmcost.Price{override},
		}
		svc := New(adapter, slog.Default())

		result, err := svc.ListPrices(context.Background(), llmcost.ListPricesInput{
			Namespace: ns,
			Page:      pagination.NewPage(1, 20),
			Source:    &filters.StringFilter{Eq: lo.ToPtr("system")},
		})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		// Should NOT be replaced — original system price preserved
		assert.Equal(t, systemPrice.ID, result.Items[0].ID)
		assert.Equal(t, llmcost.PriceSourceSystem, result.Items[0].Source)
	})

	t.Run("source!=manual skips overlay", func(t *testing.T) {
		adapter := &mockAdapter{
			prices:    []llmcost.Price{systemPrice},
			overrides: []llmcost.Price{override},
		}
		svc := New(adapter, slog.Default())

		result, err := svc.ListPrices(context.Background(), llmcost.ListPricesInput{
			Namespace: ns,
			Page:      pagination.NewPage(1, 20),
			Source:    &filters.StringFilter{Neq: lo.ToPtr("manual")},
		})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		// Should NOT be replaced
		assert.Equal(t, systemPrice.ID, result.Items[0].ID)
		assert.Equal(t, llmcost.PriceSourceSystem, result.Items[0].Source)
	})

	t.Run("source=manual allows overlay", func(t *testing.T) {
		adapter := &mockAdapter{
			prices:    []llmcost.Price{manualGlobalPrice},
			overrides: []llmcost.Price{}, // no override for this provider/model
		}
		svc := New(adapter, slog.Default())

		result, err := svc.ListPrices(context.Background(), llmcost.ListPricesInput{
			Namespace: ns,
			Page:      pagination.NewPage(1, 20),
			Source:    &filters.StringFilter{Eq: lo.ToPtr("manual")},
		})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		assert.Equal(t, manualGlobalPrice.ID, result.Items[0].ID)
	})

	t.Run("empty namespace skips overlay", func(t *testing.T) {
		adapter := &mockAdapter{
			prices:    []llmcost.Price{systemPrice},
			overrides: []llmcost.Price{override},
		}
		svc := New(adapter, slog.Default())

		result, err := svc.ListPrices(context.Background(), llmcost.ListPricesInput{
			Namespace: "", // no namespace
			Page:      pagination.NewPage(1, 20),
		})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		// No overlay — original preserved
		assert.Equal(t, systemPrice.ID, result.Items[0].ID)
	})

	t.Run("partial overlay replaces only matching prices", func(t *testing.T) {
		anotherSystem := makeTestPrice("anthropic", "claude-3-opus", llmcost.PriceSourceSystem)
		adapter := &mockAdapter{
			prices:    []llmcost.Price{systemPrice, anotherSystem},
			overrides: []llmcost.Price{override}, // only overrides openai/gpt-4
		}
		svc := New(adapter, slog.Default())

		result, err := svc.ListPrices(context.Background(), llmcost.ListPricesInput{
			Namespace: ns,
			Page:      pagination.NewPage(1, 20),
		})
		require.NoError(t, err)
		require.Len(t, result.Items, 2)
		// First should be overridden
		assert.Equal(t, override.ID, result.Items[0].ID)
		// Second should be unchanged
		assert.Equal(t, anotherSystem.ID, result.Items[1].ID)
	})
}

func TestSourceFilterExcludesManual(t *testing.T) {
	t.Run("nil filter does not exclude", func(t *testing.T) {
		assert.False(t, sourceFilterExcludesManual(nil))
	})

	t.Run("eq=system excludes manual", func(t *testing.T) {
		assert.True(t, sourceFilterExcludesManual(&filters.StringFilter{
			Eq: lo.ToPtr("system"),
		}))
	})

	t.Run("eq=manual does not exclude", func(t *testing.T) {
		assert.False(t, sourceFilterExcludesManual(&filters.StringFilter{
			Eq: lo.ToPtr("manual"),
		}))
	})

	t.Run("neq=manual excludes manual", func(t *testing.T) {
		assert.True(t, sourceFilterExcludesManual(&filters.StringFilter{
			Neq: lo.ToPtr("manual"),
		}))
	})

	t.Run("neq=system does not exclude", func(t *testing.T) {
		assert.False(t, sourceFilterExcludesManual(&filters.StringFilter{
			Neq: lo.ToPtr("system"),
		}))
	})

	t.Run("contains filter does not exclude", func(t *testing.T) {
		assert.False(t, sourceFilterExcludesManual(&filters.StringFilter{
			Contains: lo.ToPtr("sys"),
		}))
	})
}
