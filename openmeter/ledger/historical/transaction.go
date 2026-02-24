package historical

import (
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
)

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
