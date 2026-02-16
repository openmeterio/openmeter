package adapter

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgerhistorical "github.com/openmeterio/openmeter/openmeter/ledger/historical"
	"github.com/openmeterio/openmeter/pkg/models"
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
