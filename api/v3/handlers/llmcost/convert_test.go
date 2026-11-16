package llmcost

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
)

func TestFilterSourceInListPricesParams(t *testing.T) {
	// Compile-time assertion that the generated API type includes the Source field.
	filter := &apiv3models.ListLLMCostPricesParamsFilter{
		Source: &apiv3models.StringFieldFilter{
			Eq: lo.ToPtr("system"),
		},
	}
	require.NotNil(t, filter.Source)
	assert.Equal(t, lo.ToPtr("system"), filter.Source.Eq)
}
