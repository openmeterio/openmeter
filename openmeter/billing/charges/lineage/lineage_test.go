package lineage

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
)

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
