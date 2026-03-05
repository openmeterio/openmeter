package adapter

import (
	"fmt"
	"slices"
	"strings"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/cost"
	"github.com/openmeterio/openmeter/openmeter/llmcost"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

// costResolverFunc resolves the per-unit cost for a set of group-by values.
// Returns the resolved cost (nil if not found), an optional detail message, and an error.
// A non-nil error is fatal; a nil resolved cost with a non-empty detail means pricing was unavailable.
type costResolverFunc func(groupByValues map[string]string) (*cost.ResolvedUnitCost, string, error)

// costRowAccumulator aggregates multiple usage rows into a single cost row.
// Used when internal group-by keys (added for cost resolution) need to be collapsed.
type costRowAccumulator struct {
	usage     alpacadecimal.Decimal
	totalCost alpacadecimal.Decimal
	hasCost   bool
	details   []string
	row       cost.CostQueryRow
}

// addUsage accumulates a usage row's cost into the aggregator.
func (acc *costRowAccumulator) addUsage(usage alpacadecimal.Decimal, resolved *cost.ResolvedUnitCost, detail string) {
	acc.usage = acc.usage.Add(usage)

	if resolved != nil {
		acc.hasCost = true
		acc.row.Currency = resolved.Currency
		acc.totalCost = acc.totalCost.Add(usage.Mul(resolved.Amount))
	}

	if detail != "" && !slices.Contains(acc.details, detail) {
		acc.details = append(acc.details, detail)
	}
}

// finalize produces the final CostQueryRow from the accumulated data.
func (acc *costRowAccumulator) finalize() cost.CostQueryRow {
	acc.row.Usage = acc.usage

	if acc.hasCost {
		acc.row.Cost = &acc.totalCost
	}

	if len(acc.details) > 0 {
		acc.row.Detail = strings.Join(acc.details, "; ")
	}

	return acc.row
}

// computeCostRows takes meter query rows, resolves per-row unit costs, and optionally
// aggregates across internalGroupByKeys that were added for cost resolution but not
// requested by the user.
func computeCostRows(rows []meter.MeterQueryRow, internalGroupByKeys []string, resolver costResolverFunc) ([]cost.CostQueryRow, currencyx.Code, error) {
	type cachedResult struct {
		resolved *cost.ResolvedUnitCost
		detail   string
	}
	cache := make(map[string]cachedResult)

	needsAggregation := len(internalGroupByKeys) > 0
	var aggregationKeys []string
	aggregated := make(map[string]*costRowAccumulator)

	var costRows []cost.CostQueryRow
	var currency currencyx.Code

	for _, row := range rows {
		groupByValues := make(map[string]string, len(row.GroupBy))
		for k, v := range row.GroupBy {
			if v != nil {
				groupByValues[k] = *v
			}
		}

		// Resolve unit cost with caching.
		cacheKey := buildCacheKey(groupByValues)

		var resolved *cost.ResolvedUnitCost
		var detail string

		if cached, ok := cache[cacheKey]; ok {
			resolved = cached.resolved
			detail = cached.detail
		} else {
			var err error
			resolved, detail, err = resolver(groupByValues)
			if err != nil {
				return nil, "", err
			}
			cache[cacheKey] = cachedResult{resolved: resolved, detail: detail}
		}

		if resolved == nil && detail == "" {
			continue
		}

		usage := alpacadecimal.NewFromFloat(row.Value)

		if resolved != nil {
			currency = resolved.Currency
		}

		if needsAggregation {
			externalGroupBy, externalValues := filterGroupBy(row.GroupBy, internalGroupByKeys)
			aggKey := fmt.Sprintf("%s|%s|%s|%s|%v", row.WindowStart, row.WindowEnd, lo.FromPtr(row.Subject), lo.FromPtr(row.CustomerID), externalValues)

			acc, exists := aggregated[aggKey]
			if !exists {
				acc = &costRowAccumulator{
					row: cost.CostQueryRow{
						WindowStart: row.WindowStart,
						WindowEnd:   row.WindowEnd,
						Subject:     row.Subject,
						CustomerID:  row.CustomerID,
						GroupBy:     externalGroupBy,
					},
				}
				aggregated[aggKey] = acc
				aggregationKeys = append(aggregationKeys, aggKey)
			}

			acc.addUsage(usage, resolved, detail)
		} else {
			costRows = append(costRows, buildDirectCostRow(row, usage, resolved, detail))
		}
	}

	if needsAggregation {
		costRows = make([]cost.CostQueryRow, 0, len(aggregated))
		for _, key := range aggregationKeys {
			costRows = append(costRows, aggregated[key].finalize())
		}
	}

	// Sort by cost descending; rows without cost go last.
	slices.SortFunc(costRows, func(a, b cost.CostQueryRow) int {
		switch {
		case a.Cost == nil && b.Cost == nil:
			return 0
		case a.Cost == nil:
			return 1
		case b.Cost == nil:
			return -1
		default:
			return b.Cost.Cmp(*a.Cost)
		}
	})

	return costRows, currency, nil
}

// buildDirectCostRow creates a CostQueryRow from a single meter row without aggregation.
func buildDirectCostRow(row meter.MeterQueryRow, usage alpacadecimal.Decimal, resolved *cost.ResolvedUnitCost, detail string) cost.CostQueryRow {
	costRow := cost.CostQueryRow{
		Usage:       usage,
		Detail:      detail,
		WindowStart: row.WindowStart,
		WindowEnd:   row.WindowEnd,
		Subject:     row.Subject,
		CustomerID:  row.CustomerID,
	}

	if resolved != nil {
		costRow.Currency = resolved.Currency
		c := usage.Mul(resolved.Amount)
		costRow.Cost = &c
	}

	if len(row.GroupBy) > 0 {
		costRow.GroupBy = row.GroupBy
	}

	return costRow
}

// filterGroupBy splits a group-by map into external keys (for output) and their string values (for aggregation keys),
// excluding any keys in excludeKeys.
func filterGroupBy(groupBy map[string]*string, excludeKeys []string) (filtered map[string]*string, values map[string]string) {
	for k, v := range groupBy {
		if slices.Contains(excludeKeys, k) {
			continue
		}

		if filtered == nil {
			filtered = make(map[string]*string)
		}

		filtered[k] = v
		if v != nil {
			if values == nil {
				values = make(map[string]string)
			}
			values[k] = *v
		}
	}

	return filtered, values
}

// costPerTokenForType extracts the per-token cost for the given token type from ModelPricing.
// Returns an error if the token type is unknown or the pricing has no value for the requested dimension.
func costPerTokenForType(pricing llmcost.ModelPricing, tokenType feature.LLMTokenType) (alpacadecimal.Decimal, error) {
	switch tokenType {
	case feature.LLMTokenTypeInput:
		return pricing.InputPerToken, nil
	case feature.LLMTokenTypeOutput:
		return pricing.OutputPerToken, nil
	case feature.LLMTokenTypeCacheRead:
		if pricing.CacheReadPerToken == nil {
			return alpacadecimal.Decimal{}, fmt.Errorf("no cache_read pricing available for this model")
		}
		return *pricing.CacheReadPerToken, nil
	case feature.LLMTokenTypeCacheWrite:
		if pricing.CacheWritePerToken == nil {
			return alpacadecimal.Decimal{}, fmt.Errorf("no cache_write pricing available for this model")
		}
		return *pricing.CacheWritePerToken, nil
	case feature.LLMTokenTypeReasoning:
		if pricing.ReasoningPerToken == nil {
			return alpacadecimal.Decimal{}, fmt.Errorf("no reasoning pricing available for this model")
		}
		return *pricing.ReasoningPerToken, nil
	default:
		return alpacadecimal.Decimal{}, fmt.Errorf("unknown LLM token type: %s", tokenType)
	}
}

// buildCacheKey builds a deterministic cache key for a map of group-by values.
func buildCacheKey(groupByValues map[string]string) string {
	sortedKeys := make([]string, 0, len(groupByValues))

	for k := range groupByValues {
		sortedKeys = append(sortedKeys, k)
	}
	slices.Sort(sortedKeys)

	var b strings.Builder
	for _, k := range sortedKeys {
		fmt.Fprintf(&b, "%s=%s\x00", k, groupByValues[k])
	}

	return b.String()
}
