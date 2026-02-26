package testutils

import (
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
)

type AnyEntryInput struct {
	Address     ledger.PostingAddress
	AmountValue alpacadecimal.Decimal
}

var _ ledger.EntryInput = (*AnyEntryInput)(nil)

func (a *AnyEntryInput) PostingAddress() ledger.PostingAddress {
	return a.Address
}

func (a *AnyEntryInput) Amount() alpacadecimal.Decimal {
	return a.AmountValue
}

type AnyTransactionInput struct {
	BookedAtValue     time.Time
	EntryInputsValues []*AnyEntryInput
}

var _ ledger.TransactionInput = (*AnyTransactionInput)(nil)

func (a *AnyTransactionInput) BookedAt() time.Time {
	return a.BookedAtValue
}

func (a *AnyTransactionInput) EntryInputs() []ledger.EntryInput {
	return lo.Map(a.EntryInputsValues, func(e *AnyEntryInput, _ int) ledger.EntryInput {
		return e
	})
}

func (a *AnyTransactionInput) AsGroupInput(namespace string, annotations models.Annotations) ledger.TransactionGroupInput {
	return &AnyTransactionGroupInput{NamespaceValue: namespace, TransactionsValues: []*AnyTransactionInput{a}, AnnotationsValue: annotations}
}

type AnyTransactionGroupInput struct {
	NamespaceValue     string
	TransactionsValues []*AnyTransactionInput
	AnnotationsValue   models.Annotations
}

var _ ledger.TransactionGroupInput = (*AnyTransactionGroupInput)(nil)

func (a *AnyTransactionGroupInput) Namespace() string {
	return a.NamespaceValue
}

func (a *AnyTransactionGroupInput) Transactions() []ledger.TransactionInput {
	return lo.Map(a.TransactionsValues, func(t *AnyTransactionInput, _ int) ledger.TransactionInput {
		return t
	})
}

func (a *AnyTransactionGroupInput) Annotations() models.Annotations {
	return a.AnnotationsValue
}
