package llmcost

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
)

func TestFilterSourceInListPricesParams(t *testing.T) {
	// Compile-time assertion that the generated API type includes the Source field.
	filter := &api.ListLLMCostPricesParamsFilter{
		Source: &api.StringFieldFilter{
			Eq: lo.ToPtr("system"),
		},
	}
	require.NotNil(t, filter.Source)
	assert.Equal(t, lo.ToPtr("system"), filter.Source.Eq)
}
