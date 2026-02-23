package historical

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
)

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

// ----------------------------------------------------------------------------
// Implementation specific methods
// ----------------------------------------------------------------------------

func (t *TransactionInput) Validate(ctx context.Context) error {
	// Let's validate that the entries add up
	if err := ledger.ValidateInvariance(ctx, lo.Map(t.entryInputs, func(e *EntryInput, _ int) ledger.EntryInput {
		return e
	})); err != nil {
		return err
	}

	// Let's validate routing
	if err := ledger.ValidateRouting(ctx, lo.Map(t.entryInputs, func(e *EntryInput, _ int) ledger.EntryInput {
		return e
	})); err != nil {
		return err
	}

	// Let's validate the entries themselves
	for _, entry := range t.entryInputs {
		if err := ledger.ValidateEntryInput(ctx, entry); err != nil {
			return fmt.Errorf("invalid entry: %w", err)
		}
	}

	return nil
}

type TransactionGroupInput struct {
	namespace    string
	transactions []*TransactionInput
	annotations  models.Annotations
}

var _ ledger.TransactionGroupInput = (*TransactionGroupInput)(nil)

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

type Transaction struct {
	data    TransactionData
	entries []*Entry
}

func NewTransactionFromData(data TransactionData, entries []EntryData) *Transaction {
	return &Transaction{
		data: data,
		entries: lo.Map(entries, func(e EntryData, _ int) *Entry {
			return &Entry{data: e}
		}),
	}
}

// ----------------------------------------------------------------------------
// Let's implement ledger.Transaction interface
// ----------------------------------------------------------------------------

var _ ledger.Transaction = (*Transaction)(nil)

// From TransactionInput

func (t *Transaction) TransactionInput() ledger.TransactionInput {
	return &TransactionInput{
		bookedAt: t.data.BookedAt,
		entryInputs: lo.Map(t.entries, func(e *Entry, _ int) *EntryInput {
			return &EntryInput{
				amount:  e.data.Amount,
				address: e.PostingAddress(),
			}
		}),
	}
}

func (t *Transaction) EntryInputs() []ledger.EntryInput {
	return t.TransactionInput().EntryInputs()
}

func (t *Transaction) AsGroupInput(namespace string, annotations models.Annotations) ledger.TransactionGroupInput {
	return t.TransactionInput().AsGroupInput(namespace, annotations)
}

// From Transaction

func (t *Transaction) Entries() []ledger.Entry {
	return lo.Map(t.entries, func(e *Entry, _ int) ledger.Entry {
		return e
	})
}

func (t *Transaction) ID() models.NamespacedID {
	return models.NamespacedID{
		Namespace: t.data.Namespace,
		ID:        t.data.ID,
	}
}

func (t *Transaction) BookedAt() time.Time {
	return t.data.BookedAt
}

type TransactionGroup struct {
	data         TransactionGroupData
	transactions []*Transaction
}

var _ ledger.TransactionGroup = (*TransactionGroup)(nil)

func (t *TransactionGroup) Transactions() []ledger.Transaction {
	return lo.Map(t.transactions, func(t *Transaction, _ int) ledger.Transaction {
		return t
	})
}

func (t *TransactionGroup) Annotations() models.Annotations {
	return t.data.Annotations
}
