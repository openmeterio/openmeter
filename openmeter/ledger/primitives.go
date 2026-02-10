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

// Contains routing information for the account
type Address interface {
	models.Equaler[Address]

	ID() models.NamespacedID
	Type() AccountType
	Dimensions() []Dimension
}

type Balance interface {
	Settled() alpacadecimal.Decimal
	Pending() alpacadecimal.Decimal
}

// Account represents a ledger account.
type Account interface {
	Address() Address

	// SubAccount returns a new account that is a sub-account of the current account defined by the given dimensions.
	SubAccount(...Dimension) (Account, error)

	// Gets the current account balance
	GetBalance(ctx context.Context) (Balance, error)

	// ListEntries(ctx context.Context, params any) (pagination.Result[LedgerEntry], error)
}

type Dimension interface {
	models.Equaler[Dimension]

	Key() string // can be typed but str should be fine
	Value() any
}

// ----------------------------------------------------------------------------
// Transaction primitives
// ----------------------------------------------------------------------------

type EntryInput interface {
	Account() Address
	Amount() alpacadecimal.Decimal
}

type Entry interface {
	EntryInput
	TransactionID() models.NamespacedID
}

type TransactionInput interface {
	BookedAt() time.Time
	EntryInputs() []EntryInput
	AsGroupInput(annotations models.Annotations) TransactionGroupInput
}

// Transaction represents a list of entries booked at the same time
type Transaction interface {
	TransactionInput
	Entries() []Entry
	ID() models.NamespacedID
}

type TransactionGroupInput interface {
	Transactions() []TransactionInput
	Annotations() models.Annotations
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

	// SetUpTransactionInput is a no-op that runs some validations and returns a TransactionInput object that can be committed later
	SetUpTransactionInput(ctx context.Context, at time.Time, entries []EntryInput) (TransactionInput, error)

	// CommitGroup commits a list of transactions on the Ledger atomically
	CommitGroup(ctx context.Context, group TransactionGroupInput) (TransactionGroup, error)

	// // ListTransactions lists transactions on the Ledger according to some filters
	// //
	// // TODO: Cursoring gets problematic due to diff between wall_clock and booked_at. It would be convenient to return in order of booked_at as that simplifies parsing. This API will likely change.
	// ListTransactions(ctx context.Context, params ListTransactionsInput) (pagination.Result[Transaction], error)
}

type ListTransactionsInput struct {
	Cursor *pagination.Cursor
	Limit  int

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
