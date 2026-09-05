package adapter_test

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/llmcost"
	llmcostadapter "github.com/openmeterio/openmeter/openmeter/llmcost/adapter"
	"github.com/openmeterio/openmeter/openmeter/testutils"
)

func newLLMCostTestAdapter(t *testing.T) (llmcost.Adapter, *db.Client) {
	t.Helper()

	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEntMigrated)
	dbClient := testDB.EntDriver.Client()

	t.Cleanup(func() {
		_ = dbClient.Close()
		testDB.Close(t)
	})

	adapter, err := llmcostadapter.New(llmcostadapter.Config{
		Client: dbClient,
		Logger: testutils.NewDiscardLogger(t),
	})
	require.NoError(t, err)

	return adapter, dbClient
}

func TestAdapterResolvePricePrefersNamespaceOverride(t *testing.T) {
	ctx := context.Background()
	adapter, _ := newLLMCostTestAdapter(t)

	effectiveFrom := time.Now().Add(-time.Hour)

	globalPrice := llmcost.Price{
		Provider:  "openai",
		ModelID:   "gpt-test",
		ModelName: "GPT Test",
		Pricing: llmcost.ModelPricing{
			InputPerToken:  alpacadecimal.NewFromFloat(0.001),
			OutputPerToken: alpacadecimal.NewFromFloat(0.002),
		},
		Currency:      "USD",
		Source:        llmcost.PriceSourceSystem,
		EffectiveFrom: effectiveFrom,
	}
	require.NoError(t, adapter.UpsertGlobalPrice(ctx, globalPrice))

	override, err := adapter.CreateOverride(ctx, llmcost.CreateOverrideInput{
		Namespace: "ns-1",
		Provider:  "openai",
		ModelID:   "gpt-test",
		ModelName: "GPT Test",
		Pricing: llmcost.ModelPricing{
			InputPerToken:  alpacadecimal.NewFromFloat(0.010),
			OutputPerToken: alpacadecimal.NewFromFloat(0.020),
		},
		Currency:      "USD",
		EffectiveFrom: effectiveFrom,
	})
	require.NoError(t, err)

	resolved, err := adapter.ResolvePrice(ctx, llmcost.ResolvePriceInput{
		Namespace: "ns-1",
		Provider:  "openai",
		ModelID:   "gpt-test",
	})
	require.NoError(t, err)
	require.Equal(t, override.ID, resolved.ID, "namespace override must win over the global price")

	resolvedGlobal, err := adapter.ResolvePrice(ctx, llmcost.ResolvePriceInput{
		Namespace: "ns-2",
		Provider:  "openai",
		ModelID:   "gpt-test",
	})
	require.NoError(t, err)
	require.Equal(t, resolvedGlobal.Pricing.InputPerToken.String(), globalPrice.Pricing.InputPerToken.String(), "namespaces without an override resolve to the global price")
}
