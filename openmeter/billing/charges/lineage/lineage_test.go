package lineage

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
)

// TestBackfillAdvanceLineageSegmentsInputValidateCurrency documents that
// advance-lineage backfill is currency agnostic: fiat and custom currencies
// are tracked identically, since lineage only records credit provenance and
// never performs currency-specific accounting itself.
func TestBackfillAdvanceLineageSegmentsInputValidateCurrency(t *testing.T) {
	baseInput := BackfillAdvanceLineageSegmentsInput{
		Namespace:                 "test-namespace",
		CustomerID:                "test-customer",
		Amount:                    alpacadecimal.NewFromInt(10),
		BackingTransactionGroupID: "test-transaction-group",
	}

	t.Run("fiat currency", func(t *testing.T) {
		input := baseInput
		input.Currency = currenciestestutils.NewFiatCurrency(t, "USD")

		require.NoError(t, input.Validate())
	})

	t.Run("custom currency", func(t *testing.T) {
		input := baseInput
		input.Currency = currenciestestutils.NewCustomCurrency(t, "CREDITS", 2)

		require.NoError(t, input.Validate())
	})

	t.Run("missing currency", func(t *testing.T) {
		input := baseInput

		err := input.Validate()
		require.Error(t, err)
		require.ErrorContains(t, err, "currency is required")
	})
}

func TestSegmentValidateRequiresSourceBackingTransactionGroupForAdvanceBackfilledSource(t *testing.T) {
	sourceState := creditrealization.LineageSegmentStateAdvanceBackfilled
	backingTransactionGroupID := "recognition-txg"

	err := Segment{
		Amount:                    alpacadecimal.NewFromInt(10),
		State:                     creditrealization.LineageSegmentStateEarningsRecognized,
		BackingTransactionGroupID: &backingTransactionGroupID,
		SourceState:               &sourceState,
	}.Validate()

	require.Error(t, err)
	require.ErrorContains(t, err, "source backing transaction group id is required when source state is advance_backfilled")
}

func TestFeatureFiltersMatchAdvance(t *testing.T) {
	require.True(t, FeatureFiltersMatchAdvance(nil, nil))
	require.True(t, FeatureFiltersMatchAdvance(nil, []string{"api-calls"}))
	require.True(t, FeatureFiltersMatchAdvance([]string{"api-calls"}, []string{"api-calls"}))
	require.True(t, FeatureFiltersMatchAdvance([]string{"api-calls", "storage"}, []string{"storage"}))

	require.False(t, FeatureFiltersMatchAdvance([]string{"api-calls"}, nil))
	require.False(t, FeatureFiltersMatchAdvance([]string{"api-calls"}, []string{"storage"}))
}
