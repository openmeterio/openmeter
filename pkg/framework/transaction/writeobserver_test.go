package transaction_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	transactiontestutils "github.com/openmeterio/openmeter/pkg/framework/transaction/testutils"
)

func TestWriteObserver(t *testing.T) {
	attempt := transaction.WriteAttempt{
		Kind:      transaction.WriteKindMutation,
		Target:    "Customer",
		Operation: "OpCreate",
	}

	t.Run("writes are not reported without an observer", func(t *testing.T) {
		require.NoError(t, transaction.ReportWrite(t.Context(), attempt))
	})

	t.Run("writes reach the attached observer", func(t *testing.T) {
		recorder := &transactiontestutils.WriteRecorder{}

		ctx, err := transaction.WithWriteObserver(t.Context(), recorder)
		require.NoError(t, err)

		require.NoError(t, transaction.ReportWrite(ctx, attempt))
		require.Equal(t, []transaction.WriteAttempt{attempt}, recorder.Attempts())
	})

	t.Run("the observer can reject the write", func(t *testing.T) {
		rejected := errors.New("writes are not allowed")

		ctx, err := transaction.WithWriteObserver(t.Context(), &transactiontestutils.WriteRecorder{Reject: rejected})
		require.NoError(t, err)

		require.ErrorIs(t, transaction.ReportWrite(ctx, attempt), rejected)
	})

	t.Run("a nil observer is refused", func(t *testing.T) {
		ctx, err := transaction.WithWriteObserver(t.Context(), nil)
		require.Error(t, err)

		// the returned context is still usable, and reports to nobody
		require.NoError(t, transaction.ReportWrite(ctx, attempt))
	})
}
