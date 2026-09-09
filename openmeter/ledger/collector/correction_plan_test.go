package collector

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"
)

func TestCollectionCorrectionSelectsSourceBeforeState(t *testing.T) {
	for _, recognizedB := range []int64{0, 2, 5} {
		// Given A5 then B5, independently of how much B has been recognized.
		positions := []correctionPosition{
			{id: "A", order: 0, earnings: alpacadecimal.NewFromInt(5)},
			{id: "B", order: 1, accrued: alpacadecimal.NewFromInt(5 - recognizedB), earnings: alpacadecimal.NewFromInt(recognizedB)},
		}
		// When correcting 6, then return B5 and A1 in every state.
		selected, err := planCollectionCorrection(collectionCorrectionInput{positions: positions, amount: alpacadecimal.NewFromInt(6)})
		require.NoError(t, err)
		require.Len(t, selected, 2)
		require.Equal(t, "B", selected[0].id)
		require.Equal(t, float64(5), selected[0].amount.InexactFloat64())
		require.Equal(t, float64(recognizedB), selected[0].earnings.InexactFloat64())
		require.Equal(t, "A", selected[1].id)
		require.Equal(t, float64(1), selected[1].amount.InexactFloat64())
	}
}

func TestCollectionCorrectionUsesOriginalBackingOrder(t *testing.T) {
	// Given an older backing with a lexically newer source ID, and uncovered remainder.
	now := time.Now()
	positions := []correctionPosition{
		{id: "Z", recordedAt: now, orderKey: "first", earnings: alpacadecimal.NewFromInt(5)},
		{id: "A", recordedAt: now.Add(time.Second), orderKey: "second", accrued: alpacadecimal.NewFromInt(5)},
		{id: "uncovered", uncovered: true, accrued: alpacadecimal.NewFromInt(5)},
	}
	// When correcting 11, then backing is exhausted newest first, followed by uncovered.
	selected, err := planCollectionCorrection(collectionCorrectionInput{positions: positions, amount: alpacadecimal.NewFromInt(11)})
	require.NoError(t, err)
	require.Equal(t, "A", selected[0].id)
	require.Equal(t, "Z", selected[1].id)
	require.Equal(t, "uncovered", selected[2].id)
	require.Equal(t, float64(1), selected[2].amount.InexactFloat64())
	_, err = planCollectionCorrection(collectionCorrectionInput{positions: positions, amount: alpacadecimal.NewFromInt(16)})
	require.ErrorContains(t, err, "exceeds remaining collection balance")
}
