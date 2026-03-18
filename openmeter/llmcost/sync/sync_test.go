package sync

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/llmcost"
)

// mockFetcher implements Fetcher for testing.
type mockFetcher struct {
	source llmcost.PriceSource
	prices []llmcost.SourcePrice
	err    error
}

func (m *mockFetcher) Source() llmcost.PriceSource {
	return m.source
}

func (m *mockFetcher) Fetch(_ context.Context) ([]llmcost.SourcePrice, error) {
	return m.prices, m.err
}

func TestSyncJobRun(t *testing.T) {
	adapter := &mockAdapter{}
	logger := slog.Default()

	t.Run("fetches and reconciles from multiple sources", func(t *testing.T) {
		adapter.upsertedPrices = nil

		fetcher1 := &mockFetcher{
			source: "source_a",
			prices: []llmcost.SourcePrice{
				makePrice("source_a", "openai", "gpt-4", 0.01, 0.03),
			},
		}
		fetcher2 := &mockFetcher{
			source: "source_b",
			prices: []llmcost.SourcePrice{
				makePrice("source_b", "openai", "gpt-4", 0.01, 0.03),
			},
		}

		job := NewSyncJob(SyncJobConfig{
			Repo:     adapter,
			Logger:   logger,
			Fetchers: []Fetcher{fetcher1, fetcher2},
		})

		err := job.Run(context.Background())
		require.NoError(t, err)
		assert.Len(t, adapter.upsertedPrices, 1)
	})

	t.Run("continues on fetcher error", func(t *testing.T) {
		adapter.upsertedPrices = nil

		fetcher1 := &mockFetcher{
			source: "source_a",
			err:    errors.New("network error"),
		}
		fetcher2 := &mockFetcher{
			source: "source_b",
			prices: []llmcost.SourcePrice{
				makePrice("source_b", "openai", "gpt-4", 0.01, 0.03),
			},
		}
		fetcher3 := &mockFetcher{
			source: "source_c",
			prices: []llmcost.SourcePrice{
				makePrice("source_c", "openai", "gpt-4", 0.01, 0.03),
			},
		}

		job := NewSyncJob(SyncJobConfig{
			Repo:     adapter,
			Logger:   logger,
			Fetchers: []Fetcher{fetcher1, fetcher2, fetcher3},
		})

		err := job.Run(context.Background())
		require.NoError(t, err)
		assert.Len(t, adapter.upsertedPrices, 1)
	})

	t.Run("normalizes model IDs during sync", func(t *testing.T) {
		adapter.upsertedPrices = nil

		fetcher1 := &mockFetcher{
			source: "source_a",
			prices: []llmcost.SourcePrice{
				makePrice("source_a", "OpenAI", "GPT-4o-20241022", 0.01, 0.03),
			},
		}
		fetcher2 := &mockFetcher{
			source: "source_b",
			prices: []llmcost.SourcePrice{
				makePrice("source_b", "openai", "gpt-4o-20241022", 0.01, 0.03),
			},
		}

		job := NewSyncJob(SyncJobConfig{
			Repo:     adapter,
			Logger:   logger,
			Fetchers: []Fetcher{fetcher1, fetcher2},
		})

		err := job.Run(context.Background())
		require.NoError(t, err)
		require.Len(t, adapter.upsertedPrices, 1)

		upserted := adapter.upsertedPrices[0]
		assert.Equal(t, "openai", string(upserted.Provider))
		assert.Equal(t, "gpt-4o", upserted.ModelID)
	})

	t.Run("no fetchers produces no prices", func(t *testing.T) {
		adapter.upsertedPrices = nil

		job := NewSyncJob(SyncJobConfig{
			Repo:     adapter,
			Logger:   logger,
			Fetchers: []Fetcher{},
		})

		err := job.Run(context.Background())
		require.NoError(t, err)
		assert.Empty(t, adapter.upsertedPrices)
	})

	t.Run("keeps azure and openai as separate providers", func(t *testing.T) {
		adapter.upsertedPrices = nil

		fetcher1 := &mockFetcher{
			source: "source_a",
			prices: []llmcost.SourcePrice{
				makePrice("source_a", "azure", "gpt-4", 0.01, 0.03),
				makePrice("source_a", "openai", "gpt-4", 0.01, 0.03),
			},
		}
		fetcher2 := &mockFetcher{
			source: "source_b",
			prices: []llmcost.SourcePrice{
				makePrice("source_b", "azure", "gpt-4", 0.01, 0.03),
				makePrice("source_b", "openai", "gpt-4", 0.01, 0.03),
			},
		}

		job := NewSyncJob(SyncJobConfig{
			Repo:     adapter,
			Logger:   logger,
			Fetchers: []Fetcher{fetcher1, fetcher2},
		})

		err := job.Run(context.Background())
		require.NoError(t, err)
		require.Len(t, adapter.upsertedPrices, 2)

		providers := map[string]bool{}
		for _, p := range adapter.upsertedPrices {
			providers[string(p.Provider)] = true
		}
		assert.True(t, providers["azure"])
		assert.True(t, providers["openai"])
	})

	t.Run("filter excludes prices", func(t *testing.T) {
		adapter.upsertedPrices = nil

		fetcher1 := &mockFetcher{
			source: "source_a",
			prices: []llmcost.SourcePrice{
				makePrice("source_a", "openai", "gpt-4", 0.01, 0.03),
				makePrice("source_a", "anthropic", "claude-3-5-sonnet", 0.003, 0.015),
			},
		}
		fetcher2 := &mockFetcher{
			source: "source_b",
			prices: []llmcost.SourcePrice{
				makePrice("source_b", "openai", "gpt-4", 0.01, 0.03),
				makePrice("source_b", "anthropic", "claude-3-5-sonnet", 0.003, 0.015),
			},
		}

		// Only include openai models
		job := NewSyncJob(SyncJobConfig{
			Repo:     adapter,
			Logger:   logger,
			Fetchers: []Fetcher{fetcher1, fetcher2},
			Filter: func(p llmcost.SourcePrice) bool {
				return p.Provider == "openai"
			},
		})

		err := job.Run(context.Background())
		require.NoError(t, err)
		require.Len(t, adapter.upsertedPrices, 1)
		assert.Equal(t, "openai", string(adapter.upsertedPrices[0].Provider))
	})

	t.Run("nil filter includes all prices", func(t *testing.T) {
		adapter.upsertedPrices = nil

		fetcher1 := &mockFetcher{
			source: "source_a",
			prices: []llmcost.SourcePrice{
				makePrice("source_a", "openai", "gpt-4", 0.01, 0.03),
				makePrice("source_a", "anthropic", "claude-3-5-sonnet", 0.003, 0.015),
			},
		}
		fetcher2 := &mockFetcher{
			source: "source_b",
			prices: []llmcost.SourcePrice{
				makePrice("source_b", "openai", "gpt-4", 0.01, 0.03),
				makePrice("source_b", "anthropic", "claude-3-5-sonnet", 0.003, 0.015),
			},
		}

		job := NewSyncJob(SyncJobConfig{
			Repo:     adapter,
			Logger:   logger,
			Fetchers: []Fetcher{fetcher1, fetcher2},
		})

		err := job.Run(context.Background())
		require.NoError(t, err)
		assert.Len(t, adapter.upsertedPrices, 2)
	})

	t.Run("configurable min source agreement", func(t *testing.T) {
		adapter.upsertedPrices = nil

		fetcher1 := &mockFetcher{
			source: "source_a",
			prices: []llmcost.SourcePrice{
				makePrice("source_a", "openai", "gpt-4", 0.01, 0.03),
			},
		}

		job := NewSyncJob(SyncJobConfig{
			Repo:               adapter,
			Logger:             logger,
			Fetchers:           []Fetcher{fetcher1},
			MinSourceAgreement: 1,
		})

		err := job.Run(context.Background())
		require.NoError(t, err)
		assert.Len(t, adapter.upsertedPrices, 1)
	})
}

func TestDeduplicateSourcePrices(t *testing.T) {
	t.Run("no duplicates unchanged", func(t *testing.T) {
		prices := []llmcost.SourcePrice{
			makePrice("source_a", "openai", "gpt-4", 0.01, 0.03),
			makePrice("source_a", "anthropic", "claude-3-5-sonnet", 0.003, 0.015),
			makePrice("source_b", "openai", "gpt-4", 0.01, 0.03),
		}

		result := deduplicateSourcePrices(prices)
		assert.Len(t, result, 3)
	})

	t.Run("removes duplicates within same source", func(t *testing.T) {
		prices := []llmcost.SourcePrice{
			makePrice("source_a", "openai", "gpt-4", 0.01, 0.03),
			makePrice("source_a", "openai", "gpt-4", 0.0101, 0.0301), // duplicate after normalization
		}

		result := deduplicateSourcePrices(prices)
		assert.Len(t, result, 1)
		assert.Equal(t, "openai", string(result[0].Provider))
	})

	t.Run("keeps duplicates across different sources", func(t *testing.T) {
		prices := []llmcost.SourcePrice{
			makePrice("source_a", "openai", "gpt-4", 0.01, 0.03),
			makePrice("source_b", "openai", "gpt-4", 0.01, 0.03),
		}

		result := deduplicateSourcePrices(prices)
		assert.Len(t, result, 2)
	})

	t.Run("prefers model name without provider prefix", func(t *testing.T) {
		p1 := makePrice("source_a", "openai", "gpt-4", 0.01, 0.03)
		p1.ModelName = "azure/gpt-4"

		p2 := makePrice("source_a", "openai", "gpt-4", 0.01, 0.03)
		p2.ModelName = "GPT-4"

		// First entry has prefix, second doesn't — should pick second
		result := deduplicateSourcePrices([]llmcost.SourcePrice{p1, p2})
		require.Len(t, result, 1)
		assert.Equal(t, "GPT-4", result[0].ModelName)
	})

	t.Run("keeps first entry when both have clean names", func(t *testing.T) {
		p1 := makePrice("source_a", "openai", "gpt-4", 0.01, 0.03)
		p1.ModelName = "GPT-4"

		p2 := makePrice("source_a", "openai", "gpt-4", 0.0101, 0.0301)
		p2.ModelName = "GPT-4 Turbo"

		result := deduplicateSourcePrices([]llmcost.SourcePrice{p1, p2})
		require.Len(t, result, 1)
		assert.Equal(t, "GPT-4", result[0].ModelName)
	})

	t.Run("keeps first entry when both have prefixed names", func(t *testing.T) {
		p1 := makePrice("source_a", "openai", "gpt-4", 0.01, 0.03)
		p1.ModelName = "azure/gpt-4"

		p2 := makePrice("source_a", "openai", "gpt-4", 0.0101, 0.0301)
		p2.ModelName = "azure_ai/gpt-4"

		result := deduplicateSourcePrices([]llmcost.SourcePrice{p1, p2})
		require.Len(t, result, 1)
		assert.Equal(t, "azure/gpt-4", result[0].ModelName)
	})

	t.Run("empty input returns empty", func(t *testing.T) {
		result := deduplicateSourcePrices(nil)
		assert.Empty(t, result)
	})

	t.Run("multiple models with mixed duplicates", func(t *testing.T) {
		prices := []llmcost.SourcePrice{
			makePrice("source_a", "openai", "gpt-4", 0.01, 0.03),
			makePrice("source_a", "openai", "gpt-4o", 0.005, 0.015),
			makePrice("source_a", "openai", "gpt-4", 0.0101, 0.0301), // dup of gpt-4
			makePrice("source_a", "anthropic", "claude-3-5-sonnet", 0.003, 0.015),
			makePrice("source_a", "openai", "gpt-4o", 0.005, 0.015), // dup of gpt-4o
		}

		result := deduplicateSourcePrices(prices)
		assert.Len(t, result, 3) // gpt-4, gpt-4o, claude-3-5-sonnet
	})
}

func TestSyncJobDeduplication(t *testing.T) {
	adapter := &mockAdapter{}
	logger := slog.Default()

	t.Run("prevents false agreement from provider normalization", func(t *testing.T) {
		adapter.upsertedPrices = nil

		// source_a returns same model under azure_ai and azure (both normalize to "azure").
		// Without dedup, the reconciler would see 2 entries from source_a and consider them
		// as agreement. With dedup, source_a contributes only 1 entry per (provider, model_id),
		// so 2 sources are still required.
		fetcher1 := &mockFetcher{
			source: "source_a",
			prices: []llmcost.SourcePrice{
				makePrice("source_a", "azure_ai", "gpt-4", 0.01, 0.03),
				makePrice("source_a", "azure", "gpt-4", 0.01, 0.03),
			},
		}
		// source_b does NOT have this model — so no agreement should be reached
		fetcher2 := &mockFetcher{
			source: "source_b",
			prices: []llmcost.SourcePrice{
				makePrice("source_b", "openai", "gpt-3.5-turbo", 0.001, 0.002),
			},
		}

		job := NewSyncJob(SyncJobConfig{
			Repo:     adapter,
			Logger:   logger,
			Fetchers: []Fetcher{fetcher1, fetcher2},
		})

		err := job.Run(context.Background())
		require.NoError(t, err)

		// azure/gpt-4 should NOT reconcile: only source_a has it (dedup collapsed azure_ai → azure)
		// openai/gpt-3.5-turbo should NOT reconcile: only source_b has it
		assert.Empty(t, adapter.upsertedPrices)
	})

	t.Run("min agreement capped at number of fetchers", func(t *testing.T) {
		adapter.upsertedPrices = nil

		fetcher := &mockFetcher{
			source: "source_a",
			prices: []llmcost.SourcePrice{
				makePrice("source_a", "openai", "gpt-4", 0.01, 0.03),
			},
		}

		// MinSourceAgreement defaults to 2, but only 1 fetcher — should cap to 1
		job := NewSyncJob(SyncJobConfig{
			Repo:     adapter,
			Logger:   logger,
			Fetchers: []Fetcher{fetcher},
		})

		err := job.Run(context.Background())
		require.NoError(t, err)
		assert.Len(t, adapter.upsertedPrices, 1)
	})
}

func TestSyncJobDefaultFetchers(t *testing.T) {
	fetchers := DefaultFetchers(nil)
	assert.GreaterOrEqual(t, len(fetchers), 1)

	sources := make([]llmcost.PriceSource, len(fetchers))
	for i, f := range fetchers {
		sources[i] = f.Source()
	}

	assert.Contains(t, sources, llmcost.PriceSource("models_dev"))
}

func TestSyncJobTolerancePassthrough(t *testing.T) {
	adapter := &mockAdapter{}
	logger := slog.Default()

	t.Run("strict tolerance rejects slight differences", func(t *testing.T) {
		adapter.upsertedPrices = nil

		fetcher1 := &mockFetcher{
			source: "source_a",
			prices: []llmcost.SourcePrice{{
				Source:   "source_a",
				Provider: "openai",
				ModelID:  "gpt-4",
				Pricing: llmcost.ModelPricing{
					InputPerToken:  alpacadecimal.NewFromFloat(0.0100),
					OutputPerToken: alpacadecimal.NewFromFloat(0.0300),
				},
			}},
		}
		fetcher2 := &mockFetcher{
			source: "source_b",
			prices: []llmcost.SourcePrice{{
				Source:   "source_b",
				Provider: "openai",
				ModelID:  "gpt-4",
				Pricing: llmcost.ModelPricing{
					InputPerToken:  alpacadecimal.NewFromFloat(0.0101),
					OutputPerToken: alpacadecimal.NewFromFloat(0.0300),
				},
			}},
		}

		// Zero tolerance: prices must match exactly
		job := NewSyncJob(SyncJobConfig{
			Repo:           adapter,
			Logger:         logger,
			Fetchers:       []Fetcher{fetcher1, fetcher2},
			PriceTolerance: 0,
		})

		err := job.Run(context.Background())
		require.NoError(t, err)
		assert.Empty(t, adapter.upsertedPrices)
	})

	t.Run("loose tolerance accepts slight differences", func(t *testing.T) {
		adapter.upsertedPrices = nil

		fetcher1 := &mockFetcher{
			source: "source_a",
			prices: []llmcost.SourcePrice{{
				Source:   "source_a",
				Provider: "openai",
				ModelID:  "gpt-4",
				Pricing: llmcost.ModelPricing{
					InputPerToken:  alpacadecimal.NewFromFloat(0.0100),
					OutputPerToken: alpacadecimal.NewFromFloat(0.0300),
				},
			}},
		}
		fetcher2 := &mockFetcher{
			source: "source_b",
			prices: []llmcost.SourcePrice{{
				Source:   "source_b",
				Provider: "openai",
				ModelID:  "gpt-4",
				Pricing: llmcost.ModelPricing{
					InputPerToken:  alpacadecimal.NewFromFloat(0.0101),
					OutputPerToken: alpacadecimal.NewFromFloat(0.0300),
				},
			}},
		}

		// 5% tolerance: should accept
		job := NewSyncJob(SyncJobConfig{
			Repo:           adapter,
			Logger:         logger,
			Fetchers:       []Fetcher{fetcher1, fetcher2},
			PriceTolerance: 0.05,
		})

		err := job.Run(context.Background())
		require.NoError(t, err)
		assert.Len(t, adapter.upsertedPrices, 1)
	})
}
