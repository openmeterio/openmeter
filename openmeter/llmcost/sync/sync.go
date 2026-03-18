package sync

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/openmeterio/openmeter/openmeter/llmcost"
)

// PriceFilterFunc is called for each source price after normalization.
// Return true to include the price, false to exclude it.
type PriceFilterFunc func(llmcost.SourcePrice) bool

// SyncJob orchestrates fetching prices from external sources,
// normalizing model IDs, and reconciling into global prices.
type SyncJob struct {
	fetchers   []Fetcher
	normalizer ModelIDNormalizer
	reconciler *Reconciler
	filter     PriceFilterFunc
	logger     *slog.Logger
}

// SyncJobConfig contains the dependencies for creating a SyncJob.
type SyncJobConfig struct {
	HTTPClient *http.Client
	Repo       llmcost.Adapter
	Logger     *slog.Logger

	// Fetchers is the list of price fetchers to use.
	// If nil, the default built-in fetchers are used.
	Fetchers []Fetcher

	// MinSourceAgreement is the minimum number of sources that must agree on a price
	// for it to be reconciled. Zero uses DefaultMinSourceAgreement.
	MinSourceAgreement int

	// PriceTolerance is the maximum allowed percentage difference (0.0–1.0) between
	// source prices for them to be considered in agreement. Negative uses DefaultPriceTolerance.
	PriceTolerance float64

	// Filter is an optional function called for each source price after normalization.
	// If set, only prices for which it returns true are included in reconciliation.
	Filter PriceFilterFunc
}

// DefaultFetchers returns the built-in price fetchers.
func DefaultFetchers(client *http.Client) []Fetcher {
	if client == nil {
		client = http.DefaultClient
	}

	return []Fetcher{
		NewModelsDevFetcher(client),
	}
}

// NewSyncJob creates a new sync job with all configured fetchers.
func NewSyncJob(config SyncJobConfig) *SyncJob {
	normalizer := NewDefaultNormalizer()

	fetchers := config.Fetchers
	if fetchers == nil {
		fetchers = DefaultFetchers(config.HTTPClient)
	}

	// Cap minAgreement at the number of fetchers — we can't require more sources
	// to agree than we actually have.
	minAgreement := config.MinSourceAgreement
	if minAgreement <= 0 {
		minAgreement = DefaultMinSourceAgreement
	}

	if numFetchers := len(fetchers); numFetchers > 0 && minAgreement > numFetchers {
		minAgreement = numFetchers
	}

	return &SyncJob{
		fetchers:   fetchers,
		normalizer: normalizer,
		reconciler: NewReconciler(config.Repo, config.Logger, minAgreement, config.PriceTolerance),
		filter:     config.Filter,
		logger:     config.Logger,
	}
}

// sourceModelKey is used to deduplicate prices from the same source after normalization.
type sourceModelKey struct {
	Source   llmcost.PriceSource
	Provider string
	ModelID  string
}

// deduplicateSourcePrices removes duplicate entries that share the same (source, provider, model_id)
// after normalization. When duplicates exist, it prefers entries whose model name does not contain
// a provider prefix (e.g., "GPT-4o" is preferred over "azure/gpt-4o").
func deduplicateSourcePrices(prices []llmcost.SourcePrice) []llmcost.SourcePrice {
	seen := make(map[sourceModelKey]int, len(prices)) // value is index into result
	result := make([]llmcost.SourcePrice, 0, len(prices))

	for _, p := range prices {
		key := sourceModelKey{Source: p.Source, Provider: string(p.Provider), ModelID: p.ModelID}

		if idx, exists := seen[key]; exists {
			// Replace if the new entry has a better model name (no provider prefix)
			if strings.Contains(result[idx].ModelName, "/") && !strings.Contains(p.ModelName, "/") {
				result[idx] = p
			}

			continue
		}

		seen[key] = len(result)
		result = append(result, p)
	}

	return result
}

// Run executes the full sync cycle: fetch → normalize → deduplicate → reconcile.
func (j *SyncJob) Run(ctx context.Context) error {
	var allPrices []llmcost.SourcePrice

	// Phase 1: Fetch from all sources and normalize
	for _, f := range j.fetchers {
		sourceName := f.Source()

		j.logger.Info("fetching prices", "source", sourceName)

		prices, err := f.Fetch(ctx)
		if err != nil {
			j.logger.Error("failed to fetch prices",
				"source", sourceName,
				"error", err)

			continue // Don't fail entire sync if one source is down
		}

		j.logger.Info("fetched prices",
			"source", sourceName,
			"count", len(prices))

		// Normalize model IDs and provider names
		for _, p := range prices {
			provider, modelID := j.normalizer.Normalize(p.ModelID, string(p.Provider))
			p.Provider = llmcost.Provider(provider)
			p.ModelID = modelID
			allPrices = append(allPrices, p)
		}
	}

	// Phase 1.5: Deduplicate within each source after normalization.
	// Provider normalization can collapse multiple raw entries (e.g., azure/gpt-4o and openai/gpt-4o)
	// into the same (source, provider, model_id) key. Without deduplication, these would create
	// false multi-source agreement in the reconciler.
	allPrices = deduplicateSourcePrices(allPrices)

	j.logger.Info("deduplicated prices", "count", len(allPrices))

	// Phase 2: Filter (optional)
	if j.filter != nil {
		filtered := allPrices[:0]
		for _, p := range allPrices {
			if j.filter(p) {
				filtered = append(filtered, p)
			}
		}

		j.logger.Info("filtered prices",
			"before", len(allPrices),
			"after", len(filtered))

		allPrices = filtered
	}

	// Phase 3: Reconcile across sources and upsert global prices
	j.logger.Info("starting reconciliation", "total_prices", len(allPrices))

	return j.reconciler.Reconcile(ctx, allPrices)
}
