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
