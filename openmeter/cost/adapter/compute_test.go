package adapter

import (
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/cost"
	"github.com/openmeterio/openmeter/openmeter/llmcost"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func strPtr(s string) *string { return &s }

func makeResolver(prices map[string]*cost.ResolvedUnitCost) costResolverFunc {
	return func(groupByValues map[string]string) (*cost.ResolvedUnitCost, string, error) {
		key := fmt.Sprint(groupByValues)
		if r, ok := prices[key]; ok {
			return r, "", nil
		}
		return nil, fmt.Sprintf("price not found for %v", groupByValues), nil
	}
}

func mustDecimal(f float64) alpacadecimal.Decimal {
	return alpacadecimal.NewFromFloat(f)
}

func TestComputeCostRows(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	windowEnd := now.Add(time.Hour)

	t.Run("no aggregation when no internal keys", func(t *testing.T) {
		// User requested all group-by dimensions — no aggregation needed.
		rows := []meter.MeterQueryRow{
			{
				Value:       100,
				WindowStart: now,
				WindowEnd:   windowEnd,
				Subject:     strPtr("alice"),
				GroupBy: map[string]*string{
					"provider":   strPtr("openai"),
					"model":      strPtr("gpt-4"),
					"token_type": strPtr("input"),
				},
			},
			{
				Value:       50,
				WindowStart: now,
				WindowEnd:   windowEnd,
				Subject:     strPtr("alice"),
				GroupBy: map[string]*string{
					"provider":   strPtr("openai"),
					"model":      strPtr("gpt-4"),
					"token_type": strPtr("output"),
				},
			},
		}

		resolver := makeResolver(map[string]*cost.ResolvedUnitCost{
			fmt.Sprint(map[string]string{"provider": "openai", "model": "gpt-4", "token_type": "input"}): {
				Amount:   mustDecimal(0.01),
				Currency: currencyx.Code(currency.USD),
			},
			fmt.Sprint(map[string]string{"provider": "openai", "model": "gpt-4", "token_type": "output"}): {
				Amount:   mustDecimal(0.03),
				Currency: currencyx.Code(currency.USD),
			},
		})

		costRows, currencyExpected, err := computeCostRows(rows, nil, resolver)
		require.NoError(t, err)
		assert.Equal(t, currencyx.Code(currency.USD), currencyExpected)
		require.Len(t, costRows, 2)

		// Sorted by cost descending: output (1.5) before input (1).
		assert.True(t, costRows[0].Usage.Equal(mustDecimal(50)))
		require.NotNil(t, costRows[0].Cost)
		assert.True(t, costRows[0].Cost.Equal(mustDecimal(1.5))) // 50 * 0.03

		assert.True(t, costRows[1].Usage.Equal(mustDecimal(100)))
		require.NotNil(t, costRows[1].Cost)
		assert.True(t, costRows[1].Cost.Equal(mustDecimal(1))) // 100 * 0.01
	})

	t.Run("aggregates across all internal keys", func(t *testing.T) {
		// User requested groupBy=["subject"] only.
		// provider, model, token_type are internal keys added for cost resolution.
		rows := []meter.MeterQueryRow{
			{
				Value:       100,
				WindowStart: now,
				WindowEnd:   windowEnd,
				Subject:     strPtr("alice"),
				GroupBy: map[string]*string{
					"provider":   strPtr("openai"),
					"model":      strPtr("gpt-4"),
					"token_type": strPtr("input"),
				},
			},
			{
				Value:       50,
				WindowStart: now,
				WindowEnd:   windowEnd,
				Subject:     strPtr("alice"),
				GroupBy: map[string]*string{
					"provider":   strPtr("openai"),
					"model":      strPtr("gpt-4"),
					"token_type": strPtr("output"),
				},
			},
		}

		resolver := makeResolver(map[string]*cost.ResolvedUnitCost{
			fmt.Sprint(map[string]string{"provider": "openai", "model": "gpt-4", "token_type": "input"}): {
				Amount:   mustDecimal(0.01),
				Currency: currencyx.Code(currency.USD),
			},
			fmt.Sprint(map[string]string{"provider": "openai", "model": "gpt-4", "token_type": "output"}): {
				Amount:   mustDecimal(0.03),
				Currency: currencyx.Code(currency.USD),
			},
		})

		internalKeys := []string{"provider", "model", "token_type"}
		costRows, currencyExpected, err := computeCostRows(rows, internalKeys, resolver)
		require.NoError(t, err)
		assert.Equal(t, currencyx.Code(currency.USD), currencyExpected)
		require.Len(t, costRows, 1)

		// usage = 100 + 50 = 150
		assert.True(t, costRows[0].Usage.Equal(mustDecimal(150)))
		assert.Equal(t, strPtr("alice"), costRows[0].Subject)

		// cost = (100 * 0.01) + (50 * 0.03) = 1 + 1.5 = 2.5
		require.NotNil(t, costRows[0].Cost)
		assert.True(t, costRows[0].Cost.Equal(mustDecimal(2.5)))

		// Internal group-by keys should be stripped
		assert.Nil(t, costRows[0].GroupBy)
	})

	t.Run("aggregates across internal keys with same unit cost", func(t *testing.T) {
		// User requested groupBy=["model","provider"], token_type is internal.
		rows := []meter.MeterQueryRow{
			{
				Value: 100, WindowStart: now, WindowEnd: windowEnd,
				GroupBy: map[string]*string{"model": strPtr("gpt-4"), "provider": strPtr("openai"), "token_type": strPtr("input")},
			},
			{
				Value: 200, WindowStart: now, WindowEnd: windowEnd,
				GroupBy: map[string]*string{"model": strPtr("gpt-4"), "provider": strPtr("openai"), "token_type": strPtr("input")},
			},
		}

		resolver := makeResolver(map[string]*cost.ResolvedUnitCost{
			fmt.Sprint(map[string]string{"model": "gpt-4", "provider": "openai", "token_type": "input"}): {
				Amount: mustDecimal(0.01), Currency: currencyx.Code(currency.USD),
			},
		})

		internalKeys := []string{"token_type"}
		costRows, _, err := computeCostRows(rows, internalKeys, resolver)
		require.NoError(t, err)
		require.Len(t, costRows, 1)

		assert.True(t, costRows[0].Usage.Equal(mustDecimal(300)))
		require.NotNil(t, costRows[0].Cost)
		assert.True(t, costRows[0].Cost.Equal(mustDecimal(3))) // 300 * 0.01

		// External keys preserved
		require.NotNil(t, costRows[0].GroupBy)
		assert.Equal(t, strPtr("gpt-4"), costRows[0].GroupBy["model"])
		assert.Equal(t, strPtr("openai"), costRows[0].GroupBy["provider"])
		_, hasTokenType := costRows[0].GroupBy["token_type"]
		assert.False(t, hasTokenType)
	})

	t.Run("aggregates per subject across internal keys", func(t *testing.T) {
		// Two subjects, each with input+output rows
		rows := []meter.MeterQueryRow{
			{
				Value: 100, WindowStart: now, WindowEnd: windowEnd, Subject: strPtr("alice"),
				GroupBy: map[string]*string{"provider": strPtr("openai"), "model": strPtr("gpt-4"), "token_type": strPtr("input")},
			},
			{
				Value: 50, WindowStart: now, WindowEnd: windowEnd, Subject: strPtr("alice"),
				GroupBy: map[string]*string{"provider": strPtr("openai"), "model": strPtr("gpt-4"), "token_type": strPtr("output")},
			},
			{
				Value: 200, WindowStart: now, WindowEnd: windowEnd, Subject: strPtr("bob"),
				GroupBy: map[string]*string{"provider": strPtr("anthropic"), "model": strPtr("claude-3-5-sonnet"), "token_type": strPtr("input")},
			},
			{
				Value: 80, WindowStart: now, WindowEnd: windowEnd, Subject: strPtr("bob"),
				GroupBy: map[string]*string{"provider": strPtr("anthropic"), "model": strPtr("claude-3-5-sonnet"), "token_type": strPtr("output")},
			},
		}

		resolver := makeResolver(map[string]*cost.ResolvedUnitCost{
			fmt.Sprint(map[string]string{"provider": "openai", "model": "gpt-4", "token_type": "input"}):                 {Amount: mustDecimal(0.01), Currency: currencyx.Code(currency.USD)},
			fmt.Sprint(map[string]string{"provider": "openai", "model": "gpt-4", "token_type": "output"}):                {Amount: mustDecimal(0.03), Currency: currencyx.Code(currency.USD)},
			fmt.Sprint(map[string]string{"provider": "anthropic", "model": "claude-3-5-sonnet", "token_type": "input"}):  {Amount: mustDecimal(0.003), Currency: currencyx.Code(currency.USD)},
			fmt.Sprint(map[string]string{"provider": "anthropic", "model": "claude-3-5-sonnet", "token_type": "output"}): {Amount: mustDecimal(0.015), Currency: currencyx.Code(currency.USD)},
		})

		internalKeys := []string{"provider", "model", "token_type"}
		costRows, _, err := computeCostRows(rows, internalKeys, resolver)
		require.NoError(t, err)
		require.Len(t, costRows, 2)

		// alice: usage=150, cost = 100*0.01 + 50*0.03 = 2.5
		assert.True(t, costRows[0].Usage.Equal(mustDecimal(150)))
		assert.Equal(t, strPtr("alice"), costRows[0].Subject)
		require.NotNil(t, costRows[0].Cost)
		assert.True(t, costRows[0].Cost.Equal(mustDecimal(2.5)))

		// bob: usage=280, cost = 200*0.003 + 80*0.015 = 0.6 + 1.2 = 1.8
		assert.True(t, costRows[1].Usage.Equal(mustDecimal(280)))
		assert.Equal(t, strPtr("bob"), costRows[1].Subject)
		require.NotNil(t, costRows[1].Cost)
		assert.True(t, costRows[1].Cost.Equal(mustDecimal(1.8)))
	})

	t.Run("preserves external group-by keys in aggregated rows", func(t *testing.T) {
		// User requested groupBy=["region"]. provider and token_type are internal.
		rows := []meter.MeterQueryRow{
			{
				Value: 100, WindowStart: now, WindowEnd: windowEnd,
				GroupBy: map[string]*string{"region": strPtr("us"), "provider": strPtr("openai"), "token_type": strPtr("input")},
			},
			{
				Value: 50, WindowStart: now, WindowEnd: windowEnd,
				GroupBy: map[string]*string{"region": strPtr("us"), "provider": strPtr("openai"), "token_type": strPtr("output")},
			},
			{
				Value: 75, WindowStart: now, WindowEnd: windowEnd,
				GroupBy: map[string]*string{"region": strPtr("eu"), "provider": strPtr("openai"), "token_type": strPtr("input")},
			},
		}

		resolver := makeResolver(map[string]*cost.ResolvedUnitCost{
			fmt.Sprint(map[string]string{"region": "us", "provider": "openai", "token_type": "input"}):  {Amount: mustDecimal(0.01), Currency: currencyx.Code(currency.USD)},
			fmt.Sprint(map[string]string{"region": "us", "provider": "openai", "token_type": "output"}): {Amount: mustDecimal(0.03), Currency: currencyx.Code(currency.USD)},
			fmt.Sprint(map[string]string{"region": "eu", "provider": "openai", "token_type": "input"}):  {Amount: mustDecimal(0.01), Currency: currencyx.Code(currency.USD)},
		})

		internalKeys := []string{"provider", "token_type"}
		costRows, _, err := computeCostRows(rows, internalKeys, resolver)
		require.NoError(t, err)
		require.Len(t, costRows, 2)

		// us: usage=150, cost = 100*0.01 + 50*0.03 = 2.5
		assert.True(t, costRows[0].Usage.Equal(mustDecimal(150)))
		require.NotNil(t, costRows[0].GroupBy)
		assert.Equal(t, strPtr("us"), costRows[0].GroupBy["region"])
		// Internal keys should be stripped
		_, hasProvider := costRows[0].GroupBy["provider"]
		_, hasTokenType := costRows[0].GroupBy["token_type"]
		assert.False(t, hasProvider)
		assert.False(t, hasTokenType)

		// eu: usage=75, cost = 75*0.01 = 0.75
		assert.True(t, costRows[1].Usage.Equal(mustDecimal(75)))
		require.NotNil(t, costRows[1].GroupBy)
		assert.Equal(t, strPtr("eu"), costRows[1].GroupBy["region"])
	})

	t.Run("aggregates across windows separately", func(t *testing.T) {
		window1End := now.Add(time.Hour)
		window2Start := window1End
		window2End := window2Start.Add(time.Hour)

		rows := []meter.MeterQueryRow{
			{
				Value: 100, WindowStart: now, WindowEnd: window1End, Subject: strPtr("alice"),
				GroupBy: map[string]*string{"provider": strPtr("openai"), "token_type": strPtr("input")},
			},
			{
				Value: 50, WindowStart: now, WindowEnd: window1End, Subject: strPtr("alice"),
				GroupBy: map[string]*string{"provider": strPtr("openai"), "token_type": strPtr("output")},
			},
			{
				Value: 200, WindowStart: window2Start, WindowEnd: window2End, Subject: strPtr("alice"),
				GroupBy: map[string]*string{"provider": strPtr("openai"), "token_type": strPtr("input")},
			},
		}

		resolver := makeResolver(map[string]*cost.ResolvedUnitCost{
			fmt.Sprint(map[string]string{"provider": "openai", "token_type": "input"}):  {Amount: mustDecimal(0.01), Currency: currencyx.Code(currency.USD)},
			fmt.Sprint(map[string]string{"provider": "openai", "token_type": "output"}): {Amount: mustDecimal(0.03), Currency: currencyx.Code(currency.USD)},
		})

		internalKeys := []string{"provider", "token_type"}
		costRows, _, err := computeCostRows(rows, internalKeys, resolver)
		require.NoError(t, err)
		require.Len(t, costRows, 2)

		// Window 1: usage=150, cost = 100*0.01 + 50*0.03 = 2.5
		assert.Equal(t, now, costRows[0].WindowStart)
		assert.Equal(t, window1End, costRows[0].WindowEnd)
		assert.True(t, costRows[0].Usage.Equal(mustDecimal(150)))
		require.NotNil(t, costRows[0].Cost)
		assert.True(t, costRows[0].Cost.Equal(mustDecimal(2.5)))

		// Window 2: usage=200, cost = 200*0.01 = 2
		assert.Equal(t, window2Start, costRows[1].WindowStart)
		assert.Equal(t, window2End, costRows[1].WindowEnd)
		assert.True(t, costRows[1].Usage.Equal(mustDecimal(200)))
		require.NotNil(t, costRows[1].Cost)
		assert.True(t, costRows[1].Cost.Equal(mustDecimal(2)))
	})

	t.Run("price not found emits detail on aggregated row", func(t *testing.T) {
		rows := []meter.MeterQueryRow{
			{
				Value: 100, WindowStart: now, WindowEnd: windowEnd, Subject: strPtr("alice"),
				GroupBy: map[string]*string{"provider": strPtr("unknown"), "token_type": strPtr("input")},
			},
		}

		resolver := func(groupByValues map[string]string) (*cost.ResolvedUnitCost, string, error) {
			return nil, "price not found for unknown/input", nil
		}

		internalKeys := []string{"provider", "token_type"}
		costRows, _, err := computeCostRows(rows, internalKeys, resolver)
		require.NoError(t, err)
		require.Len(t, costRows, 1)

		assert.True(t, costRows[0].Usage.Equal(mustDecimal(100)))
		assert.Nil(t, costRows[0].Cost)
		assert.Contains(t, costRows[0].Detail, "price not found")
	})

	t.Run("mixed resolved and unresolved in same aggregation group", func(t *testing.T) {
		rows := []meter.MeterQueryRow{
			{
				Value: 100, WindowStart: now, WindowEnd: windowEnd, Subject: strPtr("alice"),
				GroupBy: map[string]*string{"provider": strPtr("openai"), "token_type": strPtr("input")},
			},
			{
				Value: 50, WindowStart: now, WindowEnd: windowEnd, Subject: strPtr("alice"),
				GroupBy: map[string]*string{"provider": strPtr("openai"), "token_type": strPtr("output")},
			},
			{
				Value: 30, WindowStart: now, WindowEnd: windowEnd, Subject: strPtr("alice"),
				GroupBy: map[string]*string{"provider": strPtr("openai"), "token_type": strPtr("reasoning")},
			},
		}

		resolver := func(groupByValues map[string]string) (*cost.ResolvedUnitCost, string, error) {
			switch groupByValues["token_type"] {
			case "input":
				return &cost.ResolvedUnitCost{Amount: mustDecimal(0.01), Currency: currencyx.Code(currency.USD)}, "", nil
			case "output":
				return &cost.ResolvedUnitCost{Amount: mustDecimal(0.03), Currency: currencyx.Code(currency.USD)}, "", nil
			default:
				return nil, "no reasoning pricing available", nil
			}
		}

		internalKeys := []string{"provider", "token_type"}
		costRows, _, err := computeCostRows(rows, internalKeys, resolver)
		require.NoError(t, err)
		require.Len(t, costRows, 1)

		// usage = 100 + 50 + 30 = 180
		assert.True(t, costRows[0].Usage.Equal(mustDecimal(180)))
		// cost = 100*0.01 + 50*0.03 = 2.5 (reasoning has no cost, adds usage but not cost)
		require.NotNil(t, costRows[0].Cost)
		assert.True(t, costRows[0].Cost.Equal(mustDecimal(2.5)))
		// Detail should contain the unavailable pricing message
		assert.Contains(t, costRows[0].Detail, "no reasoning pricing available")
	})

	t.Run("resolver error propagates", func(t *testing.T) {
		rows := []meter.MeterQueryRow{
			{
				Value: 100, WindowStart: now, WindowEnd: windowEnd,
				GroupBy: map[string]*string{"provider": strPtr("openai"), "token_type": strPtr("input")},
			},
		}

		resolver := func(groupByValues map[string]string) (*cost.ResolvedUnitCost, string, error) {
			return nil, "", fmt.Errorf("database connection lost")
		}

		_, _, err := computeCostRows(rows, []string{"provider", "token_type"}, resolver)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "database connection lost")
	})

	t.Run("empty rows returns empty result", func(t *testing.T) {
		resolver := makeResolver(nil)
		costRows, currency, err := computeCostRows(nil, []string{"provider"}, resolver)
		require.NoError(t, err)
		assert.Empty(t, costRows)
		assert.Empty(t, currency)
	})

	t.Run("no aggregation preserves all group-by keys in output", func(t *testing.T) {
		rows := []meter.MeterQueryRow{
			{
				Value: 100, WindowStart: now, WindowEnd: windowEnd, Subject: strPtr("alice"),
				GroupBy: map[string]*string{
					"provider":   strPtr("openai"),
					"model":      strPtr("gpt-4"),
					"token_type": strPtr("input"),
				},
			},
		}

		resolver := makeResolver(map[string]*cost.ResolvedUnitCost{
			fmt.Sprint(map[string]string{"provider": "openai", "model": "gpt-4", "token_type": "input"}): {
				Amount: mustDecimal(0.01), Currency: currencyx.Code(currency.USD),
			},
		})

		// No internal keys — all group-by keys were user-requested
		costRows, _, err := computeCostRows(rows, nil, resolver)
		require.NoError(t, err)
		require.Len(t, costRows, 1)

		require.NotNil(t, costRows[0].GroupBy)
		gb := costRows[0].GroupBy
		assert.Equal(t, strPtr("openai"), gb["provider"])
		assert.Equal(t, strPtr("gpt-4"), gb["model"])
		assert.Equal(t, strPtr("input"), gb["token_type"])
		require.NotNil(t, costRows[0].Cost)
		assert.True(t, costRows[0].Cost.Equal(mustDecimal(1))) // 100 * 0.01
	})

	t.Run("partial internal keys only strips internal ones", func(t *testing.T) {
		// User requested groupBy=["provider"]. Only token_type is internal.
		rows := []meter.MeterQueryRow{
			{
				Value: 100, WindowStart: now, WindowEnd: windowEnd,
				GroupBy: map[string]*string{"provider": strPtr("openai"), "token_type": strPtr("input")},
			},
			{
				Value: 50, WindowStart: now, WindowEnd: windowEnd,
				GroupBy: map[string]*string{"provider": strPtr("openai"), "token_type": strPtr("output")},
			},
			{
				Value: 80, WindowStart: now, WindowEnd: windowEnd,
				GroupBy: map[string]*string{"provider": strPtr("anthropic"), "token_type": strPtr("input")},
			},
		}

		resolver := makeResolver(map[string]*cost.ResolvedUnitCost{
			fmt.Sprint(map[string]string{"provider": "openai", "token_type": "input"}):    {Amount: mustDecimal(0.01), Currency: currencyx.Code(currency.USD)},
			fmt.Sprint(map[string]string{"provider": "openai", "token_type": "output"}):   {Amount: mustDecimal(0.03), Currency: currencyx.Code(currency.USD)},
			fmt.Sprint(map[string]string{"provider": "anthropic", "token_type": "input"}): {Amount: mustDecimal(0.003), Currency: currencyx.Code(currency.USD)},
		})

		// Only token_type is internal
		internalKeys := []string{"token_type"}
		costRows, _, err := computeCostRows(rows, internalKeys, resolver)
		require.NoError(t, err)
		require.Len(t, costRows, 2)

		// openai: usage=150, cost = 100*0.01 + 50*0.03 = 2.5
		require.NotNil(t, costRows[0].GroupBy)
		assert.Equal(t, strPtr("openai"), costRows[0].GroupBy["provider"])
		_, hasTokenType := costRows[0].GroupBy["token_type"]
		assert.False(t, hasTokenType)
		assert.True(t, costRows[0].Usage.Equal(mustDecimal(150)))
		require.NotNil(t, costRows[0].Cost)
		assert.True(t, costRows[0].Cost.Equal(mustDecimal(2.5)))

		// anthropic: usage=80, cost = 80*0.003 = 0.24
		assert.Equal(t, strPtr("anthropic"), costRows[1].GroupBy["provider"])
		assert.True(t, costRows[1].Usage.Equal(mustDecimal(80)))
		require.NotNil(t, costRows[1].Cost)
		assert.True(t, costRows[1].Cost.Equal(mustDecimal(0.24)))
	})

	t.Run("deduplicates detail messages", func(t *testing.T) {
		rows := []meter.MeterQueryRow{
			{
				Value: 100, WindowStart: now, WindowEnd: windowEnd, Subject: strPtr("alice"),
				GroupBy: map[string]*string{"provider": strPtr("unknown"), "token_type": strPtr("input")},
			},
			{
				Value: 50, WindowStart: now, WindowEnd: windowEnd, Subject: strPtr("alice"),
				GroupBy: map[string]*string{"provider": strPtr("unknown"), "token_type": strPtr("output")},
			},
		}

		callCount := 0
		resolver := func(groupByValues map[string]string) (*cost.ResolvedUnitCost, string, error) {
			callCount++
			return nil, "price not found", nil
		}

		internalKeys := []string{"provider", "token_type"}
		costRows, _, err := computeCostRows(rows, internalKeys, resolver)
		require.NoError(t, err)
		require.Len(t, costRows, 1)

		// Same detail message should appear once, not twice
		assert.Equal(t, "price not found", costRows[0].Detail)
	})

	t.Run("preserves customer ID in aggregated rows", func(t *testing.T) {
		rows := []meter.MeterQueryRow{
			{
				Value: 100, WindowStart: now, WindowEnd: windowEnd,
				Subject: strPtr("alice"), CustomerID: strPtr("cust-1"),
				GroupBy: map[string]*string{"token_type": strPtr("input")},
			},
			{
				Value: 50, WindowStart: now, WindowEnd: windowEnd,
				Subject: strPtr("alice"), CustomerID: strPtr("cust-1"),
				GroupBy: map[string]*string{"token_type": strPtr("output")},
			},
		}

		resolver := makeResolver(map[string]*cost.ResolvedUnitCost{
			fmt.Sprint(map[string]string{"token_type": "input"}):  {Amount: mustDecimal(0.01), Currency: currencyx.Code(currency.USD)},
			fmt.Sprint(map[string]string{"token_type": "output"}): {Amount: mustDecimal(0.03), Currency: currencyx.Code(currency.USD)},
		})

		costRows, _, err := computeCostRows(rows, []string{"token_type"}, resolver)
		require.NoError(t, err)
		require.Len(t, costRows, 1)

		assert.Equal(t, strPtr("cust-1"), costRows[0].CustomerID)
		assert.True(t, costRows[0].Usage.Equal(mustDecimal(150)))
	})

	t.Run("resolver result is cached per group-by combination", func(t *testing.T) {
		rows := []meter.MeterQueryRow{
			{
				Value: 100, WindowStart: now, WindowEnd: windowEnd,
				GroupBy: map[string]*string{"provider": strPtr("openai"), "token_type": strPtr("input")},
			},
			{
				Value: 200, WindowStart: now.Add(time.Hour), WindowEnd: windowEnd.Add(time.Hour),
				GroupBy: map[string]*string{"provider": strPtr("openai"), "token_type": strPtr("input")},
			},
		}

		callCount := 0
		resolver := func(groupByValues map[string]string) (*cost.ResolvedUnitCost, string, error) {
			callCount++
			return &cost.ResolvedUnitCost{Amount: mustDecimal(0.01), Currency: currencyx.Code(currency.USD)}, "", nil
		}

		_, _, err := computeCostRows(rows, nil, resolver)
		require.NoError(t, err)
		// Same group-by values should hit cache on second call
		assert.Equal(t, 1, callCount)
	})
}

func TestCostPerTokenForType(t *testing.T) {
	cached := mustDecimal(0.005)
	reasoning := mustDecimal(0.06)
	cacheWrite := mustDecimal(0.004)

	pricing := llmcost.ModelPricing{
		InputPerToken:      mustDecimal(0.01),
		OutputPerToken:     mustDecimal(0.03),
		CacheReadPerToken:  &cached,
		ReasoningPerToken:  &reasoning,
		CacheWritePerToken: &cacheWrite,
	}

	t.Run("input", func(t *testing.T) {
		amount, err := costPerTokenForType(pricing, feature.LLMTokenTypeInput)
		require.NoError(t, err)
		assert.True(t, amount.Equal(mustDecimal(0.01)))
	})

	t.Run("output", func(t *testing.T) {
		amount, err := costPerTokenForType(pricing, feature.LLMTokenTypeOutput)
		require.NoError(t, err)
		assert.True(t, amount.Equal(mustDecimal(0.03)))
	})

	t.Run("cache_read", func(t *testing.T) {
		amount, err := costPerTokenForType(pricing, feature.LLMTokenTypeCacheRead)
		require.NoError(t, err)
		assert.True(t, amount.Equal(mustDecimal(0.005)))
	})

	t.Run("cache_write", func(t *testing.T) {
		amount, err := costPerTokenForType(pricing, feature.LLMTokenTypeCacheWrite)
		require.NoError(t, err)
		assert.True(t, amount.Equal(mustDecimal(0.004)))
	})

	t.Run("reasoning", func(t *testing.T) {
		amount, err := costPerTokenForType(pricing, feature.LLMTokenTypeReasoning)
		require.NoError(t, err)
		assert.True(t, amount.Equal(mustDecimal(0.06)))
	})

	t.Run("cache_read nil returns error", func(t *testing.T) {
		p := llmcost.ModelPricing{InputPerToken: mustDecimal(0.01), OutputPerToken: mustDecimal(0.03)}
		_, err := costPerTokenForType(p, feature.LLMTokenTypeCacheRead)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cache_read")
	})

	t.Run("reasoning nil returns error", func(t *testing.T) {
		p := llmcost.ModelPricing{InputPerToken: mustDecimal(0.01), OutputPerToken: mustDecimal(0.03)}
		_, err := costPerTokenForType(p, feature.LLMTokenTypeReasoning)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "reasoning")
	})

	t.Run("cache_write nil returns error", func(t *testing.T) {
		p := llmcost.ModelPricing{InputPerToken: mustDecimal(0.01), OutputPerToken: mustDecimal(0.03)}
		_, err := costPerTokenForType(p, feature.LLMTokenTypeCacheWrite)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cache_write")
	})

	t.Run("unknown token type returns error", func(t *testing.T) {
		_, err := costPerTokenForType(pricing, feature.LLMTokenType("unknown"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown")
	})
}

func TestBuildCacheKey(t *testing.T) {
	t.Run("deterministic regardless of map iteration order", func(t *testing.T) {
		// Build the same map many times and verify key is always identical.
		keys := make(map[string]struct{})
		for i := 0; i < 100; i++ {
			m := map[string]string{
				"provider":   "openai",
				"model":      "gpt-4",
				"token_type": "input",
			}
			keys[buildCacheKey(m)] = struct{}{}
		}
		assert.Len(t, keys, 1, "cache key should be deterministic")
	})

	t.Run("different values produce different keys", func(t *testing.T) {
		k1 := buildCacheKey(map[string]string{"provider": "openai", "model": "gpt-4"})
		k2 := buildCacheKey(map[string]string{"provider": "openai", "model": "gpt-3.5-turbo"})
		assert.NotEqual(t, k1, k2)
	})

	t.Run("different keys produce different cache keys", func(t *testing.T) {
		k1 := buildCacheKey(map[string]string{"provider": "openai"})
		k2 := buildCacheKey(map[string]string{"model": "openai"})
		assert.NotEqual(t, k1, k2)
	})

	t.Run("no collision from values containing separators", func(t *testing.T) {
		// Ensure values with = or , don't collide with different key/value combos.
		k1 := buildCacheKey(map[string]string{"a": "b,c=d"})
		k2 := buildCacheKey(map[string]string{"a": "b", "c": "d"})
		assert.NotEqual(t, k1, k2)
	})

	t.Run("empty map", func(t *testing.T) {
		assert.Equal(t, "", buildCacheKey(map[string]string{}))
	})

	t.Run("single entry", func(t *testing.T) {
		key := buildCacheKey(map[string]string{"provider": "openai"})
		assert.NotEmpty(t, key)
		assert.Contains(t, key, "provider")
		assert.Contains(t, key, "openai")
	})
}

func TestResolveDimension(t *testing.T) {
	groupByValues := map[string]string{
		"provider":   "openai",
		"model":      "gpt-4",
		"token_type": "input",
	}

	t.Run("static value takes priority", func(t *testing.T) {
		val, err := resolveDimension("anthropic", "provider", groupByValues)
		require.NoError(t, err)
		assert.Equal(t, "anthropic", val)
	})

	t.Run("resolves from group-by property", func(t *testing.T) {
		val, err := resolveDimension("", "provider", groupByValues)
		require.NoError(t, err)
		assert.Equal(t, "openai", val)
	})

	t.Run("group-by property not found in values", func(t *testing.T) {
		_, err := resolveDimension("", "missing_key", groupByValues)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing_key")
	})

	t.Run("neither static nor property returns validation error", func(t *testing.T) {
		_, err := resolveDimension("", "", groupByValues)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "neither static property nor group by key is configured")
	})

	t.Run("static value with empty group-by", func(t *testing.T) {
		val, err := resolveDimension("openai", "", nil)
		require.NoError(t, err)
		assert.Equal(t, "openai", val)
	})

	t.Run("property with nil group-by map", func(t *testing.T) {
		_, err := resolveDimension("", "provider", nil)
		require.Error(t, err)
	})
}
