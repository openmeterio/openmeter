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

// PostingAddress encapsulates an address you can post-against. This is a one-to-one mapping to a SubAccount, this address format exists for routing purposes where the full sub-sccount isn't needed.
type PostingAddress interface {
	models.Equaler[PostingAddress]

	SubAccountID() string
	AccountType() AccountType
}

type Balance interface {
	Settled() alpacadecimal.Decimal
	Pending() alpacadecimal.Decimal
}

// SubAccount is an actual address you can post against. It has all required routing information provided.
// Accounts describe ownership and purpose while SubAccounts parameterize the actual posting address.
type SubAccount interface {
	// Returns the address of the sub-account
	Address() PostingAddress

	Dimensions() SubAccountDimensions
}

// QueryDimensions is the set of dimensions that can be used to query the balance of an account
type QueryDimensions struct {
	CurrencyID string

	// TaxCodeID is the ID of the tax code that the sub-account uses.
	TaxCodeID *string

	// FeatureIDs is the IDs of the features that the sub-account uses.
	FeatureIDs []string

	// CreditPriority is the priority of the funds in the sub-account
	CreditPriority *int
}

// Account represents a ledger account tying together multiple sub-accounts.
// Accounts describe ownership and purpose while SubAccounts parameterize the actual posting address.
type Account interface {
	// Balance can be queried across sub-accounts according to QueryDimensions
	GetBalance(ctx context.Context, query QueryDimensions) (Balance, error)
}

// ----------------------------------------------------------------------------
// Transaction primitives
// ----------------------------------------------------------------------------

type EntryInput interface {
	PostingAddress() PostingAddress
	Amount() alpacadecimal.Decimal
}

type Entry interface {
	EntryInput
	TransactionID() models.NamespacedID
}

type TransactionInput interface {
	BookedAt() time.Time
	EntryInputs() []EntryInput
	AsGroupInput(namespace string, annotations models.Annotations) TransactionGroupInput
}

// Transaction represents a list of entries booked at the same time
type Transaction interface {
	TransactionInput
	Entries() []Entry
	ID() models.NamespacedID
}

type TransactionGroupInput interface {
	Namespace() string
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
	// SetUpTransactionInput is a no-op that runs some validations and returns a TransactionInput object that can be committed later
	// FIXME: maybe we don't need this:
	// - This isn't an actual two step flow
	// - This cannot become an actual two step flow without locking
	// - This cannot run further validations without knowing the namespace
	SetUpTransactionInput(ctx context.Context, at time.Time, entries []EntryInput) (TransactionInput, error)

	// CommitGroup commits a list of transactions on the Ledger atomically
	CommitGroup(ctx context.Context, group TransactionGroupInput) (TransactionGroup, error)

	// ListTransactions lists transactions on the Ledger according to some filters
	//
	// TODO: Cursoring gets problematic due to diff between wall_clock and booked_at. It would be convenient to return in order of booked_at as that simplifies parsing. This API will likely change.
	ListTransactions(ctx context.Context, params ListTransactionsInput) (pagination.Result[Transaction], error)
}

type ListTransactionsInput struct {
	Namespace string
	Cursor    *pagination.Cursor
	Limit     int

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
