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
	entries     []ledger.Entry
}

var _ ledger.Transaction = (*correctionTestTransaction)(nil)

func (t *correctionTestTransaction) BookedAt() time.Time {
	return t.bookedAt
}

func (t *correctionTestTransaction) Entries() []ledger.Entry {
	return t.entries
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

type correctionTestPostingAddress struct {
	accountType  ledger.AccountType
	subAccountID string
}

func (a correctionTestPostingAddress) AccountType() ledger.AccountType { return a.accountType }
func (a correctionTestPostingAddress) SubAccountID() string            { return a.subAccountID }
func (a correctionTestPostingAddress) Route() ledger.SubAccountRoute   { return ledger.SubAccountRoute{} }
func (a correctionTestPostingAddress) Equal(other ledger.PostingAddress) bool {
	return other != nil && a.accountType == other.AccountType() && a.subAccountID == other.SubAccountID()
}

type correctionTestEntry struct {
	address ledger.PostingAddress
	amount  alpacadecimal.Decimal
	prov    ledger.Provenance
}

func (e correctionTestEntry) AssignedID() string                    { return "" }
func (e correctionTestEntry) PostingAddress() ledger.PostingAddress { return e.address }
func (e correctionTestEntry) Amount() alpacadecimal.Decimal         { return e.amount }
func (e correctionTestEntry) IdentityKey() string                   { return "" }
func (e correctionTestEntry) SchemaVersion() ledger.EntrySchemaVersion {
	return ledger.EntrySchemaVersionCurrent
}
func (e correctionTestEntry) Provenance() ledger.Provenance      { return e.prov }
func (e correctionTestEntry) Annotations() models.Annotations    { return nil }
func (e correctionTestEntry) ID() models.NamespacedID            { return models.NamespacedID{} }
func (e correctionTestEntry) TransactionID() models.NamespacedID { return models.NamespacedID{} }

func TestCorrectTransactionRejectsCorrectionDirection(t *testing.T) {
	t.Parallel()

	_, err := CorrectTransaction(t.Context(), ResolverDependencies{}, CorrectionInput{
		At:     time.Now(),
		Amount: alpacadecimal.NewFromInt(1),
		OriginalTransaction: &correctionTestTransaction{
			id: models.NamespacedID{Namespace: "ns", ID: "tx"},
			annotations: ledger.TransactionAnnotations(
				TemplateCode(TransferCustomerFBOToAccruedTemplate{}),
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
				TemplateCode(TransferCustomerReceivableToAccruedTemplate{}),
				ledger.TransactionDirectionForward,
			),
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "customer.receivable.collect correction is not implemented")
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
			templateName:  legacyTemplateNameFundCustomerReceivable,
			expectedError: "FundCustomerReceivableTemplate correction is not implemented",
		},
		{
			name:          "legacy settle means authorization status transfer",
			templateName:  legacyTemplateNameSettleCustomerReceivablePayment,
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
					annotations: models.Annotations{
						legacyAnnotationTransactionTemplateName: tt.templateName,
						ledger.AnnotationTransactionDirection:   string(ledger.TransactionDirectionForward),
					},
				},
			})
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestCorrectTransactionDispatchesLegacyTemplateNames(t *testing.T) {
	t.Parallel()

	_, err := CorrectTransaction(t.Context(), ResolverDependencies{}, CorrectionInput{
		At:     time.Now(),
		Amount: alpacadecimal.NewFromInt(1),
		OriginalTransaction: &correctionTestTransaction{
			id: models.NamespacedID{Namespace: "ns", ID: "tx"},
			annotations: models.Annotations{
				legacyAnnotationTransactionTemplateName: legacyTemplateNameTransferCustomerReceivableToAccrued,
				ledger.AnnotationTransactionDirection:   string(ledger.TransactionDirectionForward),
			},
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "customer.receivable.collect correction is not implemented")
}

func TestCorrectTransactionDoesNotFallbackToLegacyTemplateNameWhenCodeExists(t *testing.T) {
	t.Parallel()

	_, err := CorrectTransaction(t.Context(), ResolverDependencies{}, CorrectionInput{
		At:     time.Now(),
		Amount: alpacadecimal.NewFromInt(1),
		OriginalTransaction: &correctionTestTransaction{
			id: models.NamespacedID{Namespace: "ns", ID: "tx"},
			annotations: models.Annotations{
				ledger.AnnotationTransactionTemplateCode: "unknown.template",
				legacyAnnotationTransactionTemplateName:  legacyTemplateNameSettleCustomerReceivableFromPayment,
				ledger.AnnotationTransactionDirection:    string(ledger.TransactionDirectionForward),
			},
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), `unknown correction template code "unknown.template"`)
}

func (t *correctionTestTransaction) GroupID() models.NamespacedID { return t.ID() }

func TestCorrectAuthorizeCustomerReceivablePaymentTemplate(t *testing.T) {
	t.Parallel()

	authorizedAddr := correctionTestPostingAddress{
		accountType:  ledger.AccountTypeCustomerReceivable,
		subAccountID: "sub-auth",
	}
	openAddr := correctionTestPostingAddress{
		accountType:  ledger.AccountTypeCustomerReceivable,
		subAccountID: "sub-open",
	}

	tx := &correctionTestTransaction{
		id:       models.NamespacedID{Namespace: "ns", ID: "tx-auth"},
		bookedAt: time.Now(),
		annotations: ledger.TransactionAnnotations(
			TemplateCode(AuthorizeCustomerReceivablePaymentTemplate{}),
			ledger.TransactionDirectionForward,
		),
		entries: []ledger.Entry{
			correctionTestEntry{
				address: authorizedAddr,
				amount:  alpacadecimal.NewFromInt(-40),
			},
			correctionTestEntry{
				address: openAddr,
				amount:  alpacadecimal.NewFromInt(40),
			},
		},
	}

	t.Run("full correction", func(t *testing.T) {
		corrInputs, err := CorrectTransaction(t.Context(), ResolverDependencies{}, CorrectionInput{
			At:                  time.Now(),
			Amount:              alpacadecimal.NewFromInt(40),
			OriginalTransaction: tx,
		})
		require.NoError(t, err)
		require.Len(t, corrInputs, 1)

		entries := corrInputs[0].EntryInputs()
		require.Len(t, entries, 2)
		require.True(t, entries[0].PostingAddress().Equal(authorizedAddr))
		require.True(t, entries[0].Amount().Equal(alpacadecimal.NewFromInt(40)))
		require.True(t, entries[1].PostingAddress().Equal(openAddr))
		require.True(t, entries[1].Amount().Equal(alpacadecimal.NewFromInt(-40)))
	})

	t.Run("partial correction", func(t *testing.T) {
		corrInputs, err := CorrectTransaction(t.Context(), ResolverDependencies{}, CorrectionInput{
			At:                  time.Now(),
			Amount:              alpacadecimal.NewFromInt(15),
			OriginalTransaction: tx,
		})
		require.NoError(t, err)
		require.Len(t, corrInputs, 1)

		entries := corrInputs[0].EntryInputs()
		require.Len(t, entries, 2)
		require.True(t, entries[0].PostingAddress().Equal(authorizedAddr))
		require.True(t, entries[0].Amount().Equal(alpacadecimal.NewFromInt(15)))
		require.True(t, entries[1].PostingAddress().Equal(openAddr))
		require.True(t, entries[1].Amount().Equal(alpacadecimal.NewFromInt(-15)))
	})

	t.Run("exceeds original amount", func(t *testing.T) {
		_, err := CorrectTransaction(t.Context(), ResolverDependencies{}, CorrectionInput{
			At:                  time.Now(),
			Amount:              alpacadecimal.NewFromInt(50),
			OriginalTransaction: tx,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "exceeds original transaction amount")
	})

	t.Run("missing entry", func(t *testing.T) {
		brokenTx := &correctionTestTransaction{
			id:       models.NamespacedID{Namespace: "ns", ID: "tx-broken"},
			bookedAt: time.Now(),
			annotations: ledger.TransactionAnnotations(
				TemplateCode(AuthorizeCustomerReceivablePaymentTemplate{}),
				ledger.TransactionDirectionForward,
			),
			entries: []ledger.Entry{
				correctionTestEntry{
					address: authorizedAddr,
					amount:  alpacadecimal.NewFromInt(-40),
				},
			},
		}
		_, err := CorrectTransaction(t.Context(), ResolverDependencies{}, CorrectionInput{
			At:                  time.Now(),
			Amount:              alpacadecimal.NewFromInt(40),
			OriginalTransaction: brokenTx,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "requires original authorized and open receivable entries")
	})
}

func TestCorrectSettleCustomerReceivableFromPaymentTemplate(t *testing.T) {
	t.Parallel()

	washAddr := correctionTestPostingAddress{
		accountType:  ledger.AccountTypeWash,
		subAccountID: "sub-wash",
	}
	receivableAddr := correctionTestPostingAddress{
		accountType:  ledger.AccountTypeCustomerReceivable,
		subAccountID: "sub-recv",
	}

	tx := &correctionTestTransaction{
		id:       models.NamespacedID{Namespace: "ns", ID: "tx-settle"},
		bookedAt: time.Now(),
		annotations: ledger.TransactionAnnotations(
			TemplateCode(SettleCustomerReceivableFromPaymentTemplate{}),
			ledger.TransactionDirectionForward,
		),
		entries: []ledger.Entry{
			correctionTestEntry{
				address: washAddr,
				amount:  alpacadecimal.NewFromInt(-40),
			},
			correctionTestEntry{
				address: receivableAddr,
				amount:  alpacadecimal.NewFromInt(40),
			},
		},
	}

	t.Run("full correction", func(t *testing.T) {
		corrInputs, err := CorrectTransaction(t.Context(), ResolverDependencies{}, CorrectionInput{
			At:                  time.Now(),
			Amount:              alpacadecimal.NewFromInt(40),
			OriginalTransaction: tx,
		})
		require.NoError(t, err)
		require.Len(t, corrInputs, 1)

		entries := corrInputs[0].EntryInputs()
		require.Len(t, entries, 2)
		require.True(t, entries[0].PostingAddress().Equal(washAddr))
		require.True(t, entries[0].Amount().Equal(alpacadecimal.NewFromInt(40)))
		require.True(t, entries[1].PostingAddress().Equal(receivableAddr))
		require.True(t, entries[1].Amount().Equal(alpacadecimal.NewFromInt(-40)))
	})

	t.Run("partial correction", func(t *testing.T) {
		corrInputs, err := CorrectTransaction(t.Context(), ResolverDependencies{}, CorrectionInput{
			At:                  time.Now(),
			Amount:              alpacadecimal.NewFromInt(25),
			OriginalTransaction: tx,
		})
		require.NoError(t, err)
		require.Len(t, corrInputs, 1)

		entries := corrInputs[0].EntryInputs()
		require.Len(t, entries, 2)
		require.True(t, entries[0].PostingAddress().Equal(washAddr))
		require.True(t, entries[0].Amount().Equal(alpacadecimal.NewFromInt(25)))
		require.True(t, entries[1].PostingAddress().Equal(receivableAddr))
		require.True(t, entries[1].Amount().Equal(alpacadecimal.NewFromInt(-25)))
	})

	t.Run("exceeds original amount", func(t *testing.T) {
		_, err := CorrectTransaction(t.Context(), ResolverDependencies{}, CorrectionInput{
			At:                  time.Now(),
			Amount:              alpacadecimal.NewFromInt(50),
			OriginalTransaction: tx,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "exceeds original transaction amount")
	})

	t.Run("missing entry", func(t *testing.T) {
		brokenTx := &correctionTestTransaction{
			id:       models.NamespacedID{Namespace: "ns", ID: "tx-broken"},
			bookedAt: time.Now(),
			annotations: ledger.TransactionAnnotations(
				TemplateCode(SettleCustomerReceivableFromPaymentTemplate{}),
				ledger.TransactionDirectionForward,
			),
			entries: []ledger.Entry{
				correctionTestEntry{
					address: washAddr,
					amount:  alpacadecimal.NewFromInt(-40),
				},
			},
		}
		_, err := CorrectTransaction(t.Context(), ResolverDependencies{}, CorrectionInput{
			At:                  time.Now(),
			Amount:              alpacadecimal.NewFromInt(40),
			OriginalTransaction: brokenTx,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "requires original wash and receivable entries")
	})
}
