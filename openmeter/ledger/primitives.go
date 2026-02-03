package ledger

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination/v2"
)

// ----------------------------------------------------------------------------
// Account primitives
// ----------------------------------------------------------------------------

type Address interface {
	models.Equaler[Address]

	ID() models.NamespacedID
	Dimensions() []Dimension[any]
}

type Balance interface {
	Settled() alpacadecimal.Decimal
	Pending() alpacadecimal.Decimal
}

// Account represents a ledger account.
type Account interface {
	Address() Address

	// SubAccount returns a new account that is a sub-account of the current account defined by the given dimensions.
	SubAccount(...Dimension[any]) (Account, error)

	// Gets the current account balance
	GetBalance(ctx context.Context) (Balance, error)

	// ListEntries(ctx context.Context, params any) (pagination.Result[LedgerEntry], error)
}

type Dimension[TValue any] interface {
	models.Equaler[Dimension[TValue]]

	Key() string // can be typed but str should be fine
	Value() TValue
}

// ----------------------------------------------------------------------------
// Transaction primitives
// ----------------------------------------------------------------------------

type EntryType string

const (
	EntryTypeCredit EntryType = "credit"
	EntryTypeDebit  EntryType = "debit"
)

type LedgerEntryInput interface {
	Type() EntryType
	Account() Address
	Amount() alpacadecimal.Decimal
}

type LedgerEntry interface {
	LedgerEntryInput
	TransactionID() models.NamespacedID
}

// Transaction represents a list of entries booked at the same time
type Transaction interface {
	ID() models.NamespacedID
	BookedAt() time.Time
	Entries() []LedgerEntry
	AsGroup(annotations models.Annotations) TransactionGroup
}

// TransactionGroup represents a group of transactions written to the ledger at the same time
type TransactionGroup interface {
	Transactions() []Transaction
	Annotations() models.Annotations
}

// ----------------------------------------------------------------------------
// Ledger primitives
// ----------------------------------------------------------------------------

type Ledger interface {
	// GetAccount retreives an account from the Ledger
	GetAccount(ctx context.Context, address Address) (Account, error)

	// SetUpTransaction creates a new transaction on the Ledger and returns it without committing it
	SetUpTransaction(ctx context.Context, at time.Time, entries []LedgerEntryInput) (Transaction, error)

	// CommitGroup commits a list of transactions on the Ledger atomically
	CommitGroup(ctx context.Context, group TransactionGroup) error

	// ListTransactions lists transactions on the Ledger according to some filters
	//
	// TODO: Cursoring gets problematic due to diff between wall_clock and booked_at. It would be convenient to return in order of booked_at as that simplifies parsing. This API will likely change.
	ListTransactions(ctx context.Context, params ListTransactionsInput) (pagination.Result[Transaction], error)
}

type ListTransactionsInput struct {
	Cursor        *pagination.Cursor
	Limit         int
	TransactionID *models.NamespacedID
}

func (i ListTransactionsInput) Validate() error {
	if i.Limit < 1 {
		return errors.New("limit must be greater than 0")
	}

	if i.TransactionID != nil {
		if err := i.TransactionID.Validate(); err != nil {
			return fmt.Errorf("transaction id: %w", err)
		}
	}

	return nil
}
