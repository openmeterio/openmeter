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

	// Create a transaction group
	CreateTransactionGroup(ctx context.Context, transactionGroup CreateTransactionGroupInput) (TransactionGroupData, error)

	// Book a transaction
	BookTransaction(ctx context.Context, groupID models.NamespacedID, transaction *TransactionInput) (*Transaction, error)
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
