package lineage

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils/currency"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestBackfillAdvanceLineageSegmentsInputValidateCurrency(t *testing.T) {
	customCurrency, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeCustom).
		WithCode(currencyx.Code("CREDITS")).
		WithName("Credits").
		Build()
	require.NoError(t, err)

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
		input.Currency = currencies.Currency{Currency: customCurrency}

		err := input.Validate()
		require.Error(t, err)
		require.ErrorIs(t, err, meta.ErrCustomCurrencyNotSupported)
		require.ErrorContains(t, err, "advance lineage backfill")
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
