package transaction_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	transactiontestutils "github.com/openmeterio/openmeter/pkg/framework/transaction/testutils"
)

func TestReadOnlyScope(t *testing.T) {
	attempt := transaction.WriteAttempt{
		Kind:      transaction.WriteKindMutation,
		Target:    "Customer",
		Operation: "OpCreate",
	}

	t.Run("writes outside a read-only scope are not reported", func(t *testing.T) {
		require.False(t, transaction.IsReadOnly(t.Context()))
		require.NoError(t, transaction.ReportWrite(t.Context(), attempt))
	})

	t.Run("writes inside a read-only scope reach the observer", func(t *testing.T) {
		recorder := &transactiontestutils.ReadOnlyRecorder{}

		ctx, err := transaction.WithReadOnly(t.Context(), recorder)
		require.NoError(t, err)

		require.True(t, transaction.IsReadOnly(ctx))
		require.NoError(t, transaction.ReportWrite(ctx, attempt))
		require.Equal(t, []transaction.WriteAttempt{attempt}, recorder.Attempts())
	})

	t.Run("the observer can reject the write", func(t *testing.T) {
		rejected := errors.New("read-only scope")

		ctx, err := transaction.WithReadOnly(t.Context(), &transactiontestutils.ReadOnlyRecorder{Reject: rejected})
		require.NoError(t, err)

		require.ErrorIs(t, transaction.ReportWrite(ctx, attempt), rejected)
	})

	t.Run("a scope without an observer is refused", func(t *testing.T) {
		ctx, err := transaction.WithReadOnly(t.Context(), nil)
		require.Error(t, err)
		require.False(t, transaction.IsReadOnly(ctx))
	})
}
