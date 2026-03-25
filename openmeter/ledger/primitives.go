package ledger

import (
	"context"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/currencyx"
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
	Route() SubAccountRoute
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

	// Route returns the routing values of the sub-account.
	Route() Route

	// GetBalance returns the balance for this concrete sub-account route.
	GetBalance(ctx context.Context) (Balance, error)
}

// RouteFilter is the set of route fields that can be used to filter sub-accounts and query balances.
type RouteFilter struct {
	Currency currencyx.Code

	// DEFERRED: tax/feature not active yet.
	// Non-currency fields are retained for near-future expansion.
	TaxCode   *string
	Features  []string
	CostBasis *alpacadecimal.Decimal

	// CreditPriority is only meaningful for customer_fbo queries.
	CreditPriority *int

	// TransactionAuthorizationStatus is currently only meaningful for customer_receivable queries.
	// Nil means "do not filter by authorization status", not "open".
	TransactionAuthorizationStatus *TransactionAuthorizationStatus
}

// Account represents a ledger account tying together multiple sub-accounts.
// Accounts describe ownership and purpose while SubAccounts parameterize the actual posting address.
type Account interface {
	// Balance can be queried across sub-accounts according to RouteFilter
	GetBalance(ctx context.Context, query RouteFilter) (Balance, error)
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
	BookedAt() time.Time
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
	ID() models.NamespacedID
	Transactions() []Transaction
	Annotations() models.Annotations
}

// ----------------------------------------------------------------------------
// Ledger primitives
// ----------------------------------------------------------------------------

type Ledger interface {
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
		return ErrListTransactionsInputInvalid.WithAttrs(models.Attributes{
			"reason": "limit_invalid",
			"limit":  i.Limit,
		})
	}

	if i.TransactionID != nil {
		if err := i.TransactionID.Validate(); err != nil {
			return ErrListTransactionsInputInvalid.WithAttrs(models.Attributes{
				"reason":         "transaction_id_invalid",
				"transaction_id": i.TransactionID,
				"error":          err,
			})
		}
	}

	return nil
}
