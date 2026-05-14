package ledger

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransactionTemplateCodeAnnotations(t *testing.T) {
	t.Parallel()

	annotations := TransactionAnnotations("customer.receivable.issue", TransactionDirectionForward)

	code, err := TransactionTemplateCodeFromAnnotations(annotations)
	require.NoError(t, err)
	require.Equal(t, "customer.receivable.issue", code)
}
