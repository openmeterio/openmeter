package historical

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination/v2"
)

type Repo interface {
	entutils.TxCreator

	// Creates a list of entries and returns the data for the created entries
	CreateEntries(ctx context.Context, entries []CreateEntryInput) ([]EntryData, error)

	// Creates a transaction and returns the data for the created transaction
	CreateTransaction(ctx context.Context, transaction CreateTransactionInput) (TransactionData, error)

	// Creates a transaction group
	CreateTransactionGroup(ctx context.Context, transactionGroup CreateTransactionGroupInput) (TransactionGroupData, error)

	// Lists entries and returns the data for the listed entries
	ListEntries(ctx context.Context, input ListEntriesInput) (pagination.Result[EntryData], error)
}

// ----------------------------------------------------------------------------
// Parameter types
// ----------------------------------------------------------------------------

type ListEntriesInput struct {
	Cursor *pagination.Cursor
	Limit  int

	Filters ledger.Filters
	Expand  EntryExpand
}

type EntryExpand struct {
	Dimensions bool
}

// ----------------------------------------------------------------------------
// Input and output types
// ----------------------------------------------------------------------------

type CreateTransactionInput struct {
	Namespace string
	// Annotations models.Annotations

	GroupID  string
	BookedAt time.Time
}

type CreateTransactionGroupInput struct {
	Namespace string

	Annotations models.Annotations
}

type TransactionGroupData struct {
	ID        string
	Namespace string
	CreatedAt time.Time

	Annotations models.Annotations
}

type TransactionData struct {
	ID          string
	Namespace   string
	Annotations models.Annotations
	CreatedAt   time.Time

	BookedAt time.Time
}
