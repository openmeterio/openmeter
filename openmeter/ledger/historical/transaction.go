package historical

import (
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Transaction struct {
	data    TransactionData
	entries []*Entry
}

func NewTransactionFromData(data TransactionData, entries []EntryData) (*Transaction, error) {
	ents := make([]*Entry, 0, len(entries))
	for _, e := range entries {
		entry, err := newEntryFromData(e)
		if err != nil {
			return nil, fmt.Errorf("entry %s: %w", e.ID, err)
		}
		ents = append(ents, entry)
	}

	return &Transaction{
		data:    data,
		entries: ents,
	}, nil
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

func (t *Transaction) Annotations() models.Annotations {
	return t.data.Annotations
}

func (t *Transaction) Cursor() ledger.TransactionCursor {
	return ledger.TransactionCursor{
		BookedAt:  t.data.BookedAt,
		CreatedAt: t.data.CreatedAt,
		ID: models.NamespacedID{
			Namespace: t.data.Namespace,
			ID:        t.data.ID,
		},
	}
}

type TransactionGroup struct {
	data         TransactionGroupData
	transactions []*Transaction
}

func NewTransactionGroupFromData(data TransactionGroupData, transactions []*Transaction) *TransactionGroup {
	return &TransactionGroup{
		data:         data,
		transactions: transactions,
	}
}

var _ ledger.TransactionGroup = (*TransactionGroup)(nil)

func (t *TransactionGroup) ID() models.NamespacedID {
	return models.NamespacedID{
		Namespace: t.data.Namespace,
		ID:        t.data.ID,
	}
}

func (t *TransactionGroup) Transactions() []ledger.Transaction {
	return lo.Map(t.transactions, func(t *Transaction, _ int) ledger.Transaction {
		return t
	})
}

func (t *TransactionGroup) Annotations() models.Annotations {
	return t.data.Annotations
}
