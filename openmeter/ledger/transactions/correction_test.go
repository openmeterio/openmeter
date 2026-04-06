package transactions

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
)

type correctionTestTransaction struct {
	id          models.NamespacedID
	bookedAt    time.Time
	annotations models.Annotations
}

var _ ledger.Transaction = (*correctionTestTransaction)(nil)

func (t *correctionTestTransaction) BookedAt() time.Time {
	return t.bookedAt
}

func (t *correctionTestTransaction) Entries() []ledger.Entry {
	return nil
}

func (t *correctionTestTransaction) ID() models.NamespacedID {
	return t.id
}

func (t *correctionTestTransaction) Annotations() models.Annotations {
	return t.annotations
}

func TestCorrectTransactionRejectsCorrectionDirection(t *testing.T) {
	t.Parallel()

	_, err := CorrectTransaction(t.Context(), ResolverDependencies{}, CorrectionInput{
		At:     time.Now(),
		Amount: alpacadecimal.NewFromInt(1),
		OriginalTransaction: &correctionTestTransaction{
			id: models.NamespacedID{Namespace: "ns", ID: "tx"},
			annotations: ledger.TransactionAnnotations(
				templateName(TransferCustomerFBOToAccruedTemplate{}),
				ledger.TransactionDirectionCorrection,
			),
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot correct a correction transaction")
}

func TestCorrectTransactionDispatchesTemplateStub(t *testing.T) {
	t.Parallel()

	_, err := CorrectTransaction(t.Context(), ResolverDependencies{}, CorrectionInput{
		At:     time.Now(),
		Amount: alpacadecimal.NewFromInt(1),
		OriginalTransaction: &correctionTestTransaction{
			id: models.NamespacedID{Namespace: "ns", ID: "tx"},
			annotations: ledger.TransactionAnnotations(
				templateName(FundCustomerReceivableTemplate{}),
				ledger.TransactionDirectionForward,
			),
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "FundCustomerReceivableTemplate correction is not implemented")
}
