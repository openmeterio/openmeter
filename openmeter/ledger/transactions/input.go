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

func (t *TransactionInput) AsGroupInput(namespace string, annotations models.Annotations) ledger.TransactionGroupInput {
	return &TransactionGroupInput{
		namespace:    namespace,
		transactions: []*TransactionInput{t},
		annotations:  annotations,
	}
}

type TransactionGroupInput struct {
	namespace    string
	transactions []*TransactionInput
	annotations  models.Annotations
}

var _ ledger.TransactionGroupInput = (*TransactionGroupInput)(nil)

// ----------------------------------------------------------------------------
// Let's implement ledger.TransactionGroupInput interface
// ----------------------------------------------------------------------------

func (t *TransactionGroupInput) Transactions() []ledger.TransactionInput {
	return lo.Map(t.transactions, func(t *TransactionInput, _ int) ledger.TransactionInput {
		return t
	})
}

func (t *TransactionGroupInput) Annotations() models.Annotations {
	return t.annotations
}

func (t *TransactionGroupInput) Namespace() string {
	return t.namespace
}
