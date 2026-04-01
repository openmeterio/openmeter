package transactions

import (
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
)

type EntryInput struct {
	amount  alpacadecimal.Decimal
	address ledger.PostingAddress
}

// ----------------------------------------------------------------------------
// Let's implement ledger.EntryInput interface
// ----------------------------------------------------------------------------

var _ ledger.EntryInput = (*EntryInput)(nil)

func (e *EntryInput) PostingAddress() ledger.PostingAddress {
	return e.address
}

func (e *EntryInput) Amount() alpacadecimal.Decimal {
	return e.amount
}

type TransactionInput struct {
	bookedAt    time.Time
	entryInputs []*EntryInput
	annotations models.Annotations
}

// ----------------------------------------------------------------------------
// Let's implement ledger.TransactionInput interface
// ----------------------------------------------------------------------------

var _ ledger.TransactionInput = (*TransactionInput)(nil)

func (t *TransactionInput) BookedAt() time.Time {
	return t.bookedAt
}

func (t *TransactionInput) EntryInputs() []ledger.EntryInput {
	return lo.Map(t.entryInputs, func(e *EntryInput, _ int) ledger.EntryInput {
		return e
	})
}

func (t *TransactionInput) Annotations() models.Annotations {
	return t.annotations
}

func (t *TransactionInput) AsGroupInput(namespace string, annotations models.Annotations) ledger.TransactionGroupInput {
	return &TransactionGroupInput{
		namespace:    namespace,
		transactions: []ledger.TransactionInput{t},
		annotations:  annotations,
	}
}

func GroupInputs(namespace string, annotations models.Annotations, inputs ...ledger.TransactionInput) ledger.TransactionGroupInput {
	return &TransactionGroupInput{
		namespace:    namespace,
		transactions: inputs,
		annotations:  annotations,
	}
}

func WithAnnotations(input ledger.TransactionInput, annotations models.Annotations) ledger.TransactionInput {
	merged := make(models.Annotations, len(input.Annotations())+len(annotations))

	for key, value := range input.Annotations() {
		merged[key] = value
	}

	for key, value := range annotations {
		merged[key] = value
	}

	return &annotatedTransactionInput{
		TransactionInput: input,
		annotations:      merged,
	}
}

type annotatedTransactionInput struct {
	ledger.TransactionInput
	annotations models.Annotations
}

var _ ledger.TransactionInput = (*annotatedTransactionInput)(nil)

func (a *annotatedTransactionInput) Annotations() models.Annotations {
	return a.annotations
}

type TransactionGroupInput struct {
	namespace    string
	transactions []ledger.TransactionInput
	annotations  models.Annotations
}

var _ ledger.TransactionGroupInput = (*TransactionGroupInput)(nil)

// ----------------------------------------------------------------------------
// Let's implement ledger.TransactionGroupInput interface
// ----------------------------------------------------------------------------

func (t *TransactionGroupInput) Transactions() []ledger.TransactionInput {
	return t.transactions
}

func (t *TransactionGroupInput) Annotations() models.Annotations {
	return t.annotations
}

func (t *TransactionGroupInput) Namespace() string {
	return t.namespace
}
