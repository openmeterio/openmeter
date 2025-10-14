package adapter

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/openmeterio/openmeter/pkg/models"
)

type ModelCostProvider struct {
	providers map[string]Provider
}

func NewModelCostProvider() (*ModelCostProvider, error) {
	modelCostProvider := &ModelCostProvider{
		providers: make(map[string]Provider),
	}

	if err := modelCostProvider.initialize(); err != nil {
		return nil, fmt.Errorf("error initializing model cost provider: %w", err)
	}

	return modelCostProvider, nil
}

func (m *ModelCostProvider) initialize() error {
	url := "https://models.dev/api.json"

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error fetching JSON: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %w", err)
	}

	// Use a generic structure since we don’t know the schema
	var data ModelsDevResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return fmt.Errorf("error parsing JSON: %w", err)
	}

	m.providers = data

	return nil
}

// GetModelUnitCost gets the unit cost of a model for a given cost type
func (m ModelCostProvider) GetModelUnitCost(providerID string, modelID string, costType CostType) (float64, error) {
	provider, ok := m.providers[providerID]
	if !ok {
		return 0, models.NewGenericNotFoundError(fmt.Errorf("provider not found: %s", providerID))
	}

	model, ok := provider.Models[modelID]
	if !ok {
		return 0, models.NewGenericNotFoundError(fmt.Errorf("model not found: %s", modelID))
	}

	cost, ok := model.Cost[costType]
	if !ok {
		return 0, models.NewGenericNotFoundError(fmt.Errorf("cost type not found: %s", costType))
	}

	// Cost is per million tokens so we need to divide
	return cost / 1000000, nil
}

type ModelsDevResponse map[string]Provider

type Provider struct {
	Models map[string]Model `json:"models"`
}

type Model struct {
	ID   string    `json:"id"`
	Name string    `json:"name"`
	Cost ModelCost `json:"cost"`
}

type ModelCost map[CostType]float64

type CostType string

const (
	CostTypeInputToken  CostType = "input"
	CostTypeOutputToken CostType = "output"
	CostTypeCacheRead   CostType = "cache_read"
	CostTypeCacheWrite  CostType = "cache_write"
)
