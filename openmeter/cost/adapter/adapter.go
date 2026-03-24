package adapter

import (
	"context"
	"fmt"
	"slices"

	globlCurrency "github.com/invopop/gobl/currency"

	"github.com/openmeterio/openmeter/openmeter/cost"
	"github.com/openmeterio/openmeter/openmeter/llmcost"
	meterpkg "github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
)

func New(
	featureConnector feature.FeatureConnector,
	meterService meterpkg.Service,
	streamingConnector streaming.Connector,
	llmcostService llmcost.Service,
) cost.Adapter {
	return &adapter{
		featureConnector:   featureConnector,
		meterService:       meterService,
		streamingConnector: streamingConnector,
		llmcostService:     llmcostService,
	}
}

var _ cost.Adapter = (*adapter)(nil)

type adapter struct {
	featureConnector   feature.FeatureConnector
	meterService       meterpkg.Service
	streamingConnector streaming.Connector
	llmcostService     llmcost.Service
}

// llmPriceKey identifies a unique LLM pricing lookup by provider and model.
type llmPriceKey struct {
	provider string
	modelID  string
}

// llmPriceResult caches the result of a ResolvePrice call.
type llmPriceResult struct {
	price llmcost.Price
	err   error
}

// QueryFeatureCost queries the cost of a feature.
func (a *adapter) QueryFeatureCost(ctx context.Context, input cost.QueryFeatureCostInput) (*cost.CostQueryResult, error) {
	// Get feature
	feat, err := a.featureConnector.GetFeature(ctx, input.Namespace, input.FeatureID, feature.IncludeArchivedFeatureFalse)
	if err != nil {
		return nil, err
	}

	// Validate feature has a meter
	if feat.MeterID == nil {
		return nil, models.NewGenericValidationError(
			fmt.Errorf("feature %s has no meter associated", feat.Key),
		)
	}

	// Validate feature has a unit cost
	if feat.UnitCost == nil {
		return nil, models.NewGenericValidationError(
			fmt.Errorf("feature %s has no unit cost configured", feat.Key),
		)
	}

	// Get corresponding meter
	m, err := a.meterService.GetMeterByIDOrSlug(ctx, meterpkg.GetMeterInput{
		Namespace: input.Namespace,
		IDOrSlug:  *feat.MeterID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get meter: %w", err)
	}

	// Build usage query params (copy slices/maps to avoid mutating the caller's data)
	params := input.QueryParams
	params.GroupBy = slices.Clone(params.GroupBy)

	// Merge feature's MeterGroupByFilters into query.
	// Feature filters take precedence over request filters to prevent
	// callers from querying usage outside the feature's filter scope.
	if feat.MeterGroupByFilters != nil {
		merged := make(map[string]filter.FilterString, len(params.FilterGroupBy)+len(feat.MeterGroupByFilters))
		for k, v := range params.FilterGroupBy {
			merged[k] = v
		}
		for k, v := range feat.MeterGroupByFilters {
			merged[k] = v
		}
		params.FilterGroupBy = merged
	}

	// For LLM unit cost: add provider, model and token_type to GroupBy so we can resolve per-row unit costs.
	// Track which ones were added internally so we can aggregate across them if the user didn't request them.
	internalGroupByKeys := addLLMGroupByKeys(feat, &params)

	// Query usage data
	rows, err := a.streamingConnector.QueryMeter(ctx, input.Namespace, m, params)
	if err != nil {
		return nil, fmt.Errorf("failed to query meter: %w", err)
	}

	// Pre-resolve LLM prices for all unique (provider, model) pairs in the result set.
	// ResolvePrice only depends on (provider, model), not token_type, so this avoids
	// redundant calls when the same model has multiple token types (input, output, etc.).
	priceCache := a.getLLMPrices(ctx, feat, rows)

	// Calculate cost
	costRows, currency, err := computeCostRows(rows, internalGroupByKeys, func(groupByValues map[string]string) (*cost.ResolvedUnitCost, string, error) {
		resolved, err := a.resolveUnitCost(ctx, feat, groupByValues, priceCache)
		if err != nil {
			// If the error is a not found error we surface it directly as detail
			// explaining why pricing is unavailable.
			if models.IsGenericNotFoundError(err) {
				return nil, err.Error(), nil
			}
			return nil, "", fmt.Errorf("failed to resolve unit cost: %w", err)
		}
		return resolved, "", nil
	})
	if err != nil {
		return nil, err
	}

	// Set default currency if not set
	if currency == "" {
		currency = currencyx.Code(globlCurrency.USD)
	}

	return &cost.CostQueryResult{
		Currency: currency,
		Rows:     costRows,
	}, nil
}

// TODO: batch query prices from the LLM Cost Service, instead of resolving them one by one.
// getLLMPrices scans meter rows to collect all unique (provider, model) pairs and resolves their prices
func (a *adapter) getLLMPrices(ctx context.Context, feat *feature.Feature, rows []meterpkg.MeterQueryRow) map[llmPriceKey]llmPriceResult {
	if feat.UnitCost.Type != feature.UnitCostTypeLLM || feat.UnitCost.LLM == nil {
		return nil
	}

	llmConf := feat.UnitCost.LLM
	cache := make(map[llmPriceKey]llmPriceResult)

	for _, row := range rows {
		provider := llmConf.Provider

		// Resolve provider from group-by if not static.
		if provider == "" && llmConf.ProviderProperty != "" {
			if v := row.GroupBy[llmConf.ProviderProperty]; v != nil {
				provider = *v
			}
		}

		modelID := llmConf.Model

		// Resolve model from group-by if not static.
		if modelID == "" && llmConf.ModelProperty != "" {
			if v := row.GroupBy[llmConf.ModelProperty]; v != nil {
				modelID = *v
			}
		}

		// Normalize provider and model ID to match the canonical forms stored in the LLM cost database.
		provider, modelID = llmcost.NormalizeModelID(provider, modelID)

		// If the provider or model is not resolved, cache a PriceNotFoundError so the
		// downstream lookup in resolveLLMUnitCost gets a graceful error instead of a
		// fatal "price not in cache" panic.
		if provider == "" || modelID == "" {
			key := llmPriceKey{provider, modelID}
			if _, exists := cache[key]; !exists {
				cache[key] = llmPriceResult{err: llmcost.NewPriceNotFoundError(provider, modelID)}
			}

			continue
		}

		// If the price is already in the cache, skip it.
		key := llmPriceKey{provider, modelID}
		if _, exists := cache[key]; exists {
			continue
		}

		// Resolve the price otherwise.
		price, err := a.llmcostService.ResolvePrice(ctx, llmcost.ResolvePriceInput{
			Namespace: feat.Namespace,
			Provider:  llmcost.Provider(provider),
			ModelID:   modelID,
		})

		// Add the price to the cache.
		// We don't need to return the error as we surface it in the response
		cache[key] = llmPriceResult{price: price, err: err}
	}

	return cache
}

// addLLMGroupByKeys adds LLM dimension properties to the query's GroupBy if they aren't
// already present. Returns the list of keys that were added internally.
func addLLMGroupByKeys(feat *feature.Feature, params *streaming.QueryParams) []string {
	// If the feature doesn't have an LLM unit cost, return an empty list.
	if feat.UnitCost.Type != feature.UnitCostTypeLLM || feat.UnitCost.LLM == nil {
		return nil
	}

	// Get LLM configuration
	llmConf := feat.UnitCost.LLM
	props := []string{llmConf.ProviderProperty, llmConf.ModelProperty, llmConf.TokenTypeProperty}

	var internalKeys []string
	for _, prop := range props {
		if prop != "" && !slices.Contains(params.GroupBy, prop) {
			params.GroupBy = append(params.GroupBy, prop)
			internalKeys = append(internalKeys, prop)
		}
	}

	return internalKeys
}

// resolveUnitCost resolves the per-unit cost for a feature given group-by dimension values.
func (a *adapter) resolveUnitCost(ctx context.Context, feat *feature.Feature, groupByValues map[string]string, priceCache map[llmPriceKey]llmPriceResult) (*cost.ResolvedUnitCost, error) {
	if feat.UnitCost == nil {
		return nil, nil
	}

	// Resolve unit cost based on the feature's unit cost type.
	switch feat.UnitCost.Type {
	// Manual unit cost
	case feature.UnitCostTypeManual:
		if feat.UnitCost.Manual == nil {
			return nil, fmt.Errorf("feature %s has manual unit cost type but no manual configuration", feat.Key)
		}

		return &cost.ResolvedUnitCost{
			Amount:   feat.UnitCost.Manual.Amount,
			Currency: currencyx.Code(globlCurrency.USD),
		}, nil

	// LLM unit cost
	case feature.UnitCostTypeLLM:
		return a.resolveLLMUnitCost(ctx, feat, groupByValues, priceCache)

	// Unknown unit cost type
	default:
		return nil, fmt.Errorf("unknown unit cost type: %s", feat.UnitCost.Type)
	}
}

// resolveLLMUnitCost resolves LLM-based unit cost by looking up provider, model, and token type
// from either static configuration or group-by dimension values.
func (a *adapter) resolveLLMUnitCost(ctx context.Context, feat *feature.Feature, groupByValues map[string]string, priceCache map[llmPriceKey]llmPriceResult) (*cost.ResolvedUnitCost, error) {
	if feat.UnitCost.LLM == nil {
		return nil, fmt.Errorf("feature %s has LLM unit cost type but no LLM configuration", feat.Key)
	}

	if a.llmcostService == nil {
		return nil, fmt.Errorf("LLM cost service is not available")
	}

	llmConf := feat.UnitCost.LLM

	// Resolve provider
	provider, err := resolveDimension(llmConf.Provider, llmConf.ProviderProperty, groupByValues)
	if err != nil {
		return nil, fmt.Errorf("resolving provider for feature %s: %w", feat.Key, err)
	}

	// Resolve model
	modelID, err := resolveDimension(llmConf.Model, llmConf.ModelProperty, groupByValues)
	if err != nil {
		return nil, fmt.Errorf("resolving model for feature %s: %w", feat.Key, err)
	}

	// Normalize provider and model ID to match the canonical forms stored in the LLM cost database.
	provider, modelID = llmcost.NormalizeModelID(provider, modelID)

	// Resolve token type
	tokenTypeStr, err := resolveDimension(llmConf.TokenType, llmConf.TokenTypeProperty, groupByValues)
	if err != nil {
		return nil, fmt.Errorf("resolving token type for feature %s: %w", feat.Key, err)
	}

	// Look up price from pre-resolved cache.
	key := llmPriceKey{provider, modelID}
	cached, ok := priceCache[key]

	// This should never happen as it should have been pre-resolved in the cache.
	if !ok {
		return nil, fmt.Errorf("resolving LLM price for provider=%s model=%s: price not in cache", provider, modelID)
	}

	// If the price is not found, return the cached error directly.
	if cached.err != nil {
		return nil, cached.err
	}

	// Resolve token type cost
	amount, err := costPerTokenForType(cached.price.Pricing, feature.LLMTokenType(tokenTypeStr))
	if err != nil {
		return nil, fmt.Errorf("resolving token type cost for provider=%s model=%s type=%s: %w", provider, modelID, tokenTypeStr, err)
	}

	return &cost.ResolvedUnitCost{
		Amount:   amount,
		Currency: currencyx.Code(cached.price.Currency),
	}, nil
}

// resolveDimension resolves an LLM dimension value from either a static value or a group-by property.
// e.g. provider (static) vs providerProperty (groupBy key)
func resolveDimension(staticValue, groupByKey string, groupByValues map[string]string) (string, error) {
	// If the dimension has a static value, return it.
	if staticValue != "" {
		return staticValue, nil
	}

	// If the dimension has no group by key, return an error.
	if groupByKey == "" {
		return "", models.NewGenericValidationError(
			fmt.Errorf("neither static property nor group by key is configured"),
		)
	}

	// If the dimension has a property name, resolve it from the group-by values.
	value, ok := groupByValues[groupByKey]
	if !ok {
		return "", fmt.Errorf("group-by key %s not found", groupByKey)
	}

	// Return the val
	return value, nil
}
