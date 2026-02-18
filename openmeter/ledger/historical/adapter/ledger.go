package adapter

import (
	"context"
	stdsql "database/sql"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	ledgerentrydb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgerentry"
	ledgertransactiondb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgerhistorical "github.com/openmeterio/openmeter/openmeter/ledger/historical"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination/v2"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

func (r *repo) BookTransaction(ctx context.Context, groupID models.NamespacedID, input *ledgerhistorical.TransactionInput) (*ledgerhistorical.Transaction, error) {
	if input == nil {
		return nil, fmt.Errorf("transaction input is required")
	}

	entity, err := r.db.LedgerTransaction.Create().
		SetNamespace(groupID.Namespace).
		SetGroupID(groupID.ID).
		SetBookedAt(input.BookedAt()).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create ledger transaction: %w", err)
	}

	entryInputs := input.EntryInputs()
	accountTypesBySubAccountID := make(map[string]ledger.AccountType, len(entryInputs))
	createInputs := make([]*db.LedgerEntryCreate, 0, len(entryInputs))
	for _, entryInput := range entryInputs {
		subAccountID := entryInput.PostingAddress().SubAccountID()
		accountTypesBySubAccountID[subAccountID] = entryInput.PostingAddress().AccountType()

		createInputs = append(createInputs, r.db.LedgerEntry.Create().
			SetNamespace(groupID.Namespace).
			SetSubAccountID(subAccountID).
			SetAmount(entryInput.Amount()).
			SetTransactionID(entity.ID))
	}

	createdEntries := make([]*db.LedgerEntry, 0, len(createInputs))
	if len(createInputs) > 0 {
		createdEntries, err = r.db.LedgerEntry.CreateBulk(createInputs...).Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create ledger entries: %w", err)
		}
	}

	return ledgerhistorical.NewTransactionFromData(
		ledgerhistorical.TransactionData{
			ID:          entity.ID,
			Namespace:   entity.Namespace,
			Annotations: entity.Annotations,
			CreatedAt:   entity.CreatedAt,
			BookedAt:    entity.BookedAt,
		},
		lo.Map(createdEntries, func(e *db.LedgerEntry, _ int) ledgerhistorical.EntryData {
			return ledgerhistorical.EntryData{
				ID:            e.ID,
				Namespace:     e.Namespace,
				Annotations:   e.Annotations,
				CreatedAt:     e.CreatedAt,
				AccountID:     e.SubAccountID,
				AccountType:   accountTypesBySubAccountID[e.SubAccountID],
				Amount:        e.Amount,
				TransactionID: e.TransactionID,
			}
		}),
	), nil
}

func (r *repo) CreateTransactionGroup(ctx context.Context, transactionGroup ledgerhistorical.CreateTransactionGroupInput) (ledgerhistorical.TransactionGroupData, error) {
	entity, err := r.db.LedgerTransactionGroup.Create().
		SetNamespace(transactionGroup.Namespace).
		SetAnnotations(transactionGroup.Annotations).
		Save(ctx)
	if err != nil {
		return ledgerhistorical.TransactionGroupData{}, fmt.Errorf("failed to create transaction group: %w", err)
	}

	return ledgerhistorical.TransactionGroupData{
		ID:          entity.ID,
		Namespace:   entity.Namespace,
		CreatedAt:   entity.CreatedAt,
		Annotations: entity.Annotations,
	}, nil
}

func (r *repo) SumEntries(ctx context.Context, query ledger.Query) (alpacadecimal.Decimal, error) {
	q := sumEntriesQuery{
		query: query,
	}

	entryQuery := q.Build(r.db)

	var rows []struct {
		SumAmount stdsql.NullString `json:"sum_amount,omitempty"`
	}

	err := entryQuery.
		Aggregate(db.As(db.Sum(ledgerentrydb.FieldAmount), "sum_amount")).
		Scan(ctx, &rows)
	if err != nil {
		return alpacadecimal.Decimal{}, fmt.Errorf("failed to query ledger entries to sum: %w", err)
	}

	if len(rows) == 0 || !rows[0].SumAmount.Valid {
		return alpacadecimal.NewFromInt(0), nil
	}

	total, err := alpacadecimal.NewFromString(rows[0].SumAmount.String)
	if err != nil {
		return alpacadecimal.Decimal{}, fmt.Errorf("failed to parse summed amount %q: %w", rows[0].SumAmount.String, err)
	}

	return total, nil
}

func (r *repo) ListTransactions(ctx context.Context, input ledger.ListTransactionsInput) (pagination.Result[*ledgerhistorical.Transaction], error) {
	query := r.db.LedgerTransaction.Query().
		Where(ledgertransactiondb.Namespace(input.Namespace)).
		WithEntries(func(q *db.LedgerEntryQuery) {
			q.Order(
				ledgerentrydb.ByCreatedAt(),
				ledgerentrydb.ByID(),
			)
			q.WithSubAccount(func(sq *db.LedgerSubAccountQuery) {
				sq.WithAccount()
			})
		})

	if input.TransactionID != nil {
		query = query.Where(ledgertransactiondb.ID(input.TransactionID.ID))
	}

	if input.Limit > 0 {
		query = query.Limit(input.Limit)
	}

	paged, err := query.Cursor(ctx, input.Cursor)
	if err != nil {
		return pagination.Result[*ledgerhistorical.Transaction]{}, fmt.Errorf("failed to list transactions: %w", err)
	}
	if len(paged.Items) == 0 {
		return pagination.Result[*ledgerhistorical.Transaction]{
			Items: []*ledgerhistorical.Transaction{},
		}, nil
	}

	items, err := slicesx.MapWithErr(paged.Items, func(tx *db.LedgerTransaction) (*ledgerhistorical.Transaction, error) {
		entryData, err := slicesx.MapWithErr(tx.Edges.Entries, func(entry *db.LedgerEntry) (ledgerhistorical.EntryData, error) {
			subAccount, err := entry.Edges.SubAccountOrErr()
			if err != nil {
				return ledgerhistorical.EntryData{}, fmt.Errorf("entry %s missing sub-account edge: %w", entry.ID, err)
			}

			account, err := subAccount.Edges.AccountOrErr()
			if err != nil {
				return ledgerhistorical.EntryData{}, fmt.Errorf("entry %s sub-account %s missing account edge: %w", entry.ID, subAccount.ID, err)
			}

			return ledgerhistorical.EntryData{
				ID:            entry.ID,
				Namespace:     entry.Namespace,
				Annotations:   entry.Annotations,
				CreatedAt:     entry.CreatedAt,
				AccountID:     entry.SubAccountID,
				AccountType:   account.AccountType,
				Amount:        entry.Amount,
				TransactionID: entry.TransactionID,
			}, nil
		})
		if err != nil {
			return nil, fmt.Errorf("transaction %s entry hydration failed: %w", tx.ID, err)
		}

		return ledgerhistorical.NewTransactionFromData(
			ledgerhistorical.TransactionData{
				ID:          tx.ID,
				Namespace:   tx.Namespace,
				Annotations: tx.Annotations,
				CreatedAt:   tx.CreatedAt,
				BookedAt:    tx.BookedAt,
			},
			entryData,
		), nil
	})
	if err != nil {
		return pagination.Result[*ledgerhistorical.Transaction]{}, fmt.Errorf("failed to hydrate listed transactions: %w", err)
	}

	return pagination.Result[*ledgerhistorical.Transaction]{
		Items:      items,
		NextCursor: paged.NextCursor,
	}, nil
}
