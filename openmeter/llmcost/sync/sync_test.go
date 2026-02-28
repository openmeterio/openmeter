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

	t.Run("normalizes azure to openai provider", func(t *testing.T) {
		adapter.upsertedPrices = nil

		fetcher1 := &mockFetcher{
			source: "source_a",
			prices: []llmcost.SourcePrice{
				makePrice("source_a", "azure", "gpt-4", 0.01, 0.03),
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
		require.Len(t, adapter.upsertedPrices, 1)
		assert.Equal(t, "openai", string(adapter.upsertedPrices[0].Provider))
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
