package historical

import (
	"context"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/pkg/models"
)

type TransactionGroupInput struct {
	Namespace    string
	Transactions []TransactionInput
}

type TransactionInput struct {
	BookedAt time.Time
	Entries  []EntryInput
}

type TransactionEntryPostingAddress struct {
	SubAccountID  string
	TransactionID string
}

type EntryInput struct {
	Amount         alpacadecimal.Decimal
	PostingAddress TransactionEntryPostingAddress
}

type TransactionGroup struct {
	models.NamespacedID
	models.ManagedModel

	Transactions []Transaction
}

type Transaction struct {
	models.NamespacedID
	models.ManagedModel

	BookedAt time.Time
	Entries  []TransactionEntry
}

type TransactionEntry struct {
	models.NamespacedID
	models.ManagedModel

	Amount         alpacadecimal.Decimal
	PostingAddress TransactionEntryPostingAddress
}

type Ledger interface {
	// CommitGroup commits a list of transactions on the Ledger atomically
	CommitGroup(ctx context.Context, group TransactionGroupInput) (TransactionGroup, error)

	// ListTransactions lists transactions on the Ledger according to some filters
	//
	// TODO: Cursoring gets problematic due to diff between wall_clock and booked_at. It would be convenient to return in order of booked_at as that simplifies parsing. This API will likely change.
	// ListTransactions(ctx context.Context, params ListTransactionsInput) (pagination.Result[Transaction], error)
}
