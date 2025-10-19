package modelcost

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/openmeterio/openmeter/pkg/models"
)

type ModelCostProvider struct {
	config    CostProviderConfig
	providers map[string]Provider
}

type CostProviderConfig struct {
	Logger *slog.Logger
	// Timeout is the timeout before fallback to local file
	Timeout time.Duration
}

func NewModelCostProvider(config CostProviderConfig) (*ModelCostProvider, error) {
	modelCostProvider := &ModelCostProvider{
		config:    config,
		providers: make(map[string]Provider),
	}

	if err := modelCostProvider.initialize(); err != nil {
		return nil, fmt.Errorf("error initializing model cost provider: %w", err)
	}

	return modelCostProvider, nil
}

func (m *ModelCostProvider) initialize() error {
	url := "https://models.dev/api.json"
	logger := m.config.Logger.WithGroup("initialize")

	// Create HTTP client with 3-second timeout
	client := &http.Client{
		Timeout: m.config.Timeout,
	}

	resp, err := client.Get(url)
	if err != nil {
		// Fallback to local file if remote request fails or times out
		return m.loadFromLocalFile()
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("request", "error", err)
		// Fallback to local file if reading response fails
		return m.loadFromLocalFile()
	}

	// Use a generic structure since we don't know the schema
	var data ModelsDevResponse
	if err := json.Unmarshal(body, &data); err != nil {
		logger.Error("json unmarshal", "error", err)

		// Fallback to local file if JSON parsing fails
		return m.loadFromLocalFile()
	}

	m.providers = data

	return nil
}

// loadFromLocalFile loads the model cost data from the local api.json file
func (m *ModelCostProvider) loadFromLocalFile() error {
	m.config.Logger.Info("loading from local file")

	// Get the directory of the current file
	filePath := "openmeter/cost/modelcost/api.json"

	// Try to read the local file
	body, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading local api.json file: %w", err)
	}

	// Parse the JSON data
	var data ModelsDevResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return fmt.Errorf("error parsing local JSON: %w", err)
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
