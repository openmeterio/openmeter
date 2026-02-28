package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/llmcost"
)

const modelsDevAPIURL = "https://models.dev/api.json"

// modelsDevProvider represents a provider entry from models.dev.
type modelsDevProvider struct {
	ID     string                    `json:"id"`
	Name   string                    `json:"name"`
	Models map[string]modelsDevModel `json:"models"`
}

// modelsDevModel represents a model entry from models.dev.
type modelsDevModel struct {
	ID   string         `json:"id"`
	Name string         `json:"name"`
	Cost *modelsDevCost `json:"cost"`
}

type modelsDevCost struct {
	Input      *float64 `json:"input"`
	Output     *float64 `json:"output"`
	CacheRead  *float64 `json:"cache_read"`
	CacheWrite *float64 `json:"cache_write"`
	Reasoning  *float64 `json:"reasoning"`
}

type modelsDevFetcher struct {
	client *http.Client
}

func NewModelsDevFetcher(client *http.Client) Fetcher {
	return &modelsDevFetcher{client: client}
}

func (f *modelsDevFetcher) Source() llmcost.PriceSource {
	return "models_dev"
}

func (f *modelsDevFetcher) Fetch(ctx context.Context) ([]llmcost.SourcePrice, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, modelsDevAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch models.dev: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("models.dev returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var providers map[string]modelsDevProvider
	if err := json.Unmarshal(body, &providers); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	now := time.Now().UTC()
	perMillion := alpacadecimal.NewFromFloat(1_000_000)
	var prices []llmcost.SourcePrice

	for providerKey, prov := range providers {
		provider := strings.ToLower(providerKey)
		if provider == "" {
			continue
		}

		for _, model := range prov.Models {
			if model.Cost == nil || model.Cost.Input == nil || model.Cost.Output == nil {
				continue
			}

			// models.dev provides prices per million tokens, convert to per-token
			// models.dev uses "provider/model" format for IDs, strip the provider prefix
			modelID := model.ID
			if parts := strings.SplitN(modelID, "/", 2); len(parts) == 2 {
				modelID = parts[1]
			}

			sp := llmcost.SourcePrice{
				Source:    "models_dev",
				Provider:  llmcost.Provider(provider),
				ModelID:   modelID,
				ModelName: model.Name,
				Pricing: llmcost.ModelPricing{
					InputPerToken:  alpacadecimal.NewFromFloat(*model.Cost.Input).Div(perMillion),
					OutputPerToken: alpacadecimal.NewFromFloat(*model.Cost.Output).Div(perMillion),
				},
				FetchedAt: now,
			}

			if model.Cost.CacheRead != nil {
				cached := alpacadecimal.NewFromFloat(*model.Cost.CacheRead).Div(perMillion)
				sp.Pricing.InputCachedPerToken = &cached
			}

			if model.Cost.CacheWrite != nil {
				cacheWrite := alpacadecimal.NewFromFloat(*model.Cost.CacheWrite).Div(perMillion)
				sp.Pricing.CacheWritePerToken = &cacheWrite
			}

			if model.Cost.Reasoning != nil {
				reasoning := alpacadecimal.NewFromFloat(*model.Cost.Reasoning).Div(perMillion)
				sp.Pricing.ReasoningPerToken = &reasoning
			}

			prices = append(prices, sp)
		}
	}

	return prices, nil
}
