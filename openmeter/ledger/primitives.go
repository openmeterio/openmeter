package ledger

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
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

	// AccountID returns the identifier of the parent account.
	AccountID() models.NamespacedID

	// Route returns the routing values of the sub-account.
	Route() Route
}

// RouteFilter is the set of route fields that can be used to filter sub-accounts and query balances.
type RouteFilter struct {
	Currency currencyx.Code

	// DEFERRED: tax/feature not active yet.
	// Non-currency fields are retained for near-future expansion.
	TaxCode   *string
	Features  []string
	CostBasis mo.Option[*alpacadecimal.Decimal]

	// CreditPriority is only meaningful for customer_fbo queries.
	CreditPriority *int

	// TransactionAuthorizationStatus is currently only meaningful for customer_receivable queries.
	// Nil means "do not filter by authorization status", not "open".
	TransactionAuthorizationStatus *TransactionAuthorizationStatus
}

// Account represents a ledger account tying together multiple sub-accounts.
// Accounts describe ownership and purpose while SubAccounts parameterize the actual posting address.
type Account interface {
	ID() models.NamespacedID
	Type() AccountType
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
	Annotations() models.Annotations
	AsGroupInput(namespace string, annotations models.Annotations) TransactionGroupInput
}

// Transaction represents a list of entries booked at the same time
type Transaction interface {
	Cursor() TransactionCursor
	BookedAt() time.Time
	Entries() []Entry
	ID() models.NamespacedID
	Annotations() models.Annotations
}

type TransactionCursor struct {
	BookedAt  time.Time
	CreatedAt time.Time
	ID        models.NamespacedID
}

// Compare returns cursor ordering by BookedAt, then CreatedAt, then ID.
// It returns -1 if c < other, 0 if equal, and 1 if c > other.
func (c TransactionCursor) Compare(other TransactionCursor) int {
	switch {
	case c.BookedAt.Before(other.BookedAt):
		return -1
	case c.BookedAt.After(other.BookedAt):
		return 1
	}

	switch {
	case c.CreatedAt.Before(other.CreatedAt):
		return -1
	case c.CreatedAt.After(other.CreatedAt):
		return 1
	}

	return strings.Compare(c.ID.ID, other.ID.ID)
}

func (c TransactionCursor) Validate() error {
	var errs []error

	if c.BookedAt.IsZero() {
		errs = append(errs, fmt.Errorf("booked_at is zero"))
	}

	if c.CreatedAt.IsZero() {
		errs = append(errs, fmt.Errorf("created_at is zero"))
	}

	if err := c.ID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("id is invalid: %w", err))
	}

	return errors.Join(errs...)
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

	// GetTransactionGroup loads a previously committed transaction group including its transactions.
	GetTransactionGroup(ctx context.Context, id models.NamespacedID) (TransactionGroup, error)

	// ListTransactions lists transactions on the Ledger according to some filters
	//
	// TODO: Cursoring gets problematic due to diff between wall_clock and booked_at. It would be convenient to return in order of booked_at as that simplifies parsing. This API will likely change.
	ListTransactions(ctx context.Context, params ListTransactionsInput) (ListTransactionsResult, error)
}

type ListTransactionsInput struct {
	Namespace string
	Cursor    *TransactionCursor
	Before    *TransactionCursor
	Limit     int

	TransactionID *models.NamespacedID

	// AccountIDs scopes the query to transactions with entries on these accounts.
	AccountIDs []string
	Currency   *currencyx.Code

	CreditMovement ListTransactionsCreditMovement

	// AnnotationFilters matches transactions whose annotations contain all the given key-value pairs.
	AnnotationFilters map[string]string
}

type ListTransactionsResult struct {
	Items      []Transaction
	NextCursor *TransactionCursor
}

func (i ListTransactionsInput) Validate() error {
	if i.Limit < 1 {
		return ErrListTransactionsInputInvalid.WithAttrs(models.Attributes{
			"reason": "limit_invalid",
			"limit":  i.Limit,
		})
	}

	if i.Cursor != nil {
		if err := i.Cursor.Validate(); err != nil {
			return ErrListTransactionsInputInvalid.WithAttrs(models.Attributes{
				"reason": "cursor_invalid",
				"cursor": i.Cursor,
				"error":  err,
			})
		}
	}

	if i.Before != nil {
		if err := i.Before.Validate(); err != nil {
			return ErrListTransactionsInputInvalid.WithAttrs(models.Attributes{
				"reason": "before_invalid",
				"before": i.Before,
				"error":  err,
			})
		}
	}

	if i.Cursor != nil && i.Before != nil {
		return ErrListTransactionsInputInvalid.WithAttrs(models.Attributes{
			"reason": "after_before_both_set",
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

	if i.Currency != nil {
		if err := i.Currency.Validate(); err != nil {
			return ErrListTransactionsInputInvalid.WithAttrs(models.Attributes{
				"reason":   "currency_invalid",
				"currency": i.Currency,
				"error":    err,
			})
		}
	}

	switch i.CreditMovement {
	case ListTransactionsCreditMovementUnspecified, ListTransactionsCreditMovementPositive, ListTransactionsCreditMovementNegative:
	default:
		return ErrListTransactionsInputInvalid.WithAttrs(models.Attributes{
			"reason":          "credit_movement_invalid",
			"credit_movement": i.CreditMovement,
		})
	}

	return nil
}

type ListTransactionsCreditMovement uint8

const (
	ListTransactionsCreditMovementUnspecified ListTransactionsCreditMovement = iota
	ListTransactionsCreditMovementPositive
	ListTransactionsCreditMovementNegative
)
