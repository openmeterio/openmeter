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

func (t *correctionTestTransaction) Cursor() ledger.TransactionCursor {
	return ledger.TransactionCursor{
		BookedAt:  t.bookedAt,
		CreatedAt: t.bookedAt,
		ID:        t.id,
	}
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
				templateName(SettleCustomerReceivableFromPaymentTemplate{}),
				ledger.TransactionDirectionForward,
			),
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "SettleCustomerReceivableFromPaymentTemplate correction is not implemented")
}

func TestCorrectTransactionDispatchesArchivedReceivablePaymentTemplates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		templateName  string
		expectedError string
	}{
		{
			name:          "legacy fund means settlement funding",
			templateName:  templateName(FundCustomerReceivableTemplate{}),
			expectedError: "FundCustomerReceivableTemplate correction is not implemented",
		},
		{
			name:          "legacy settle means authorization status transfer",
			templateName:  templateName(SettleCustomerReceivablePaymentTemplate{}),
			expectedError: "SettleCustomerReceivablePaymentTemplate correction is not implemented",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := CorrectTransaction(t.Context(), ResolverDependencies{}, CorrectionInput{
				At:     time.Now(),
				Amount: alpacadecimal.NewFromInt(1),
				OriginalTransaction: &correctionTestTransaction{
					id: models.NamespacedID{Namespace: "ns", ID: "tx"},
					annotations: ledger.TransactionAnnotations(
						tt.templateName,
						ledger.TransactionDirectionForward,
					),
				},
			})
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.expectedError)
		})
	}
}
