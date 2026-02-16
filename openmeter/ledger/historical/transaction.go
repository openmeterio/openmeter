package historical

import (
	"context"
	"errors"
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

func (t *TransactionInput) AsGroupInput(annotations models.Annotations) ledger.TransactionGroupInput {
	return &TransactionGroupInput{
		transactions: []*TransactionInput{t},
		annotations:  annotations,
	}
}

type TransactionGroupInput struct {
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

// ----------------------------------------------------------------------------
// Implementation specific methods
// ----------------------------------------------------------------------------

func (t *TransactionInput) getNamespace() string {
	return lo.FirstOrEmpty(lo.Uniq(lo.Map(t.entryInputs, func(e *EntryInput, _ int) string {
		return e.input.Namespace
	})))
}

func (t *TransactionInput) Validate(ctx context.Context) error {
	// Let's validate the namespace is the same
	nss := lo.Uniq(lo.Map(t.entryInputs, func(e *EntryInput, _ int) string {
		return e.input.Namespace
	}))
	if len(nss) > 1 {
		return errors.New("all entries must have the same namespace")
	}

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

	return nil
}

type Transaction struct {
	data    TransactionData
	entries []*Entry
}

// ----------------------------------------------------------------------------
// Let's implement ledger.Transaction interface
// ----------------------------------------------------------------------------

var _ ledger.Transaction = (*Transaction)(nil)

// From TransactionInput

func (t *Transaction) TransactionInput() ledger.TransactionInput {
	panic("not implemented")
}

func (t *Transaction) EntryInputs() []ledger.EntryInput {
	return t.TransactionInput().EntryInputs()
}

func (t *Transaction) AsGroupInput(annotations models.Annotations) ledger.TransactionGroupInput {
	return t.TransactionInput().AsGroupInput(annotations)
}

// From Transaction

func (t *Transaction) Entries() []ledger.Entry {
	panic("not implemented")
}

func (t *Transaction) ID() models.NamespacedID {
	panic("not implemented")
}

func (t *Transaction) BookedAt() time.Time {
	panic("not implemented")
}

func (t *Transaction) AsGroup(annotations models.Annotations) ledger.TransactionGroup {
	panic("not implemented")
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
