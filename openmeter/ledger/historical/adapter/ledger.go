package adapter

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgtype"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	ledgerdimensiondb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgerdimension"
	ledgerentrydb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgerentry"
	ledgertransactiondb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	ledgeraccount "github.com/openmeterio/openmeter/openmeter/ledger/account"
	ledgerhistorical "github.com/openmeterio/openmeter/openmeter/ledger/historical"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination/v2"
)

func (r *repo) CreateEntries(ctx context.Context, entries []ledgerhistorical.CreateEntryInput) ([]ledgerhistorical.EntryData, error) {
	if len(entries) == 0 {
		return []ledgerhistorical.EntryData{}, nil
	}

	createInputs := make([]*db.LedgerEntryCreate, 0, len(entries))
	for _, entry := range entries {
		create := r.db.LedgerEntry.Create().
			SetNamespace(entry.Namespace).
			SetAccountID(entry.AccountID).
			SetAccountType(entry.AccountType).
			SetAmount(entry.Amount).
			SetTransactionID(entry.TransactionID)

		// Keep nil as DB null; empty slice means explicit empty array.
		if entry.DimensionIDs != nil {
			dimensionIDs, err := toTextArray(entry.DimensionIDs)
			if err != nil {
				return nil, fmt.Errorf("failed to encode dimension ids: %w", err)
			}

			create = create.SetDimensionIds(dimensionIDs)
		}

		createInputs = append(createInputs, create)
	}

	created, err := r.db.LedgerEntry.CreateBulk(createInputs...).Save(ctx)
	if err != nil {
		if isInvalidDimensionReferenceError(err) {
			return nil, models.NewGenericValidationError(errors.New("one or more dimension ids do not exist in the entry namespace"))
		}

		return nil, fmt.Errorf("failed to create ledger entries: %w", err)
	}

	return lo.Map(created, func(entity *db.LedgerEntry, _ int) ledgerhistorical.EntryData {
		dimensionIDs := fromTextArray(entity.DimensionIds)

		return ledgerhistorical.EntryData{
			ID:            entity.ID,
			Namespace:     entity.Namespace,
			Annotations:   entity.Annotations,
			CreatedAt:     entity.CreatedAt,
			AccountID:     entity.AccountID,
			AccountType:   entity.AccountType,
			DimensionIDs:  dimensionIDs,
			Amount:        entity.Amount,
			TransactionID: entity.TransactionID,
		}
	}), nil
}

func (r *repo) CreateTransaction(ctx context.Context, transactionInput ledgerhistorical.CreateTransactionInput) (ledgerhistorical.TransactionData, error) {
	entity, err := r.db.LedgerTransaction.Create().
		SetNamespace(transactionInput.Namespace).
		SetGroupID(transactionInput.GroupID).
		SetBookedAt(transactionInput.BookedAt).
		Save(ctx)
	if err != nil {
		return ledgerhistorical.TransactionData{}, fmt.Errorf("failed to create ledger transaction: %w", err)
	}

	return ledgerhistorical.TransactionData{
		ID:          entity.ID,
		Namespace:   entity.Namespace,
		Annotations: entity.Annotations,
		CreatedAt:   entity.CreatedAt,
		BookedAt:    entity.BookedAt,
	}, nil
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

func (r *repo) ListEntries(ctx context.Context, input ledgerhistorical.ListEntriesInput) (pagination.Result[ledgerhistorical.EntryData], error) {
	query := r.db.LedgerEntry.Query()

	if input.Filters.Account != nil {
		address := input.Filters.Account
		query = query.Where(
			ledgerentrydb.AccountID(address.ID().ID),
			ledgerentrydb.Namespace(address.ID().Namespace),
		)
	}

	if input.Filters.TransactionID != nil {
		query = query.Where(ledgerentrydb.TransactionID(*input.Filters.TransactionID))
	}

	if input.Filters.BookedAtPeriod != nil {
		transactionPredicates := make([]predicate.LedgerTransaction, 0, 2)
		if input.Filters.BookedAtPeriod.From != nil {
			transactionPredicates = append(transactionPredicates, ledgertransactiondb.BookedAtGTE(*input.Filters.BookedAtPeriod.From))
		}
		if input.Filters.BookedAtPeriod.To != nil {
			transactionPredicates = append(transactionPredicates, ledgertransactiondb.BookedAtLT(*input.Filters.BookedAtPeriod.To))
		}
		if len(transactionPredicates) > 0 {
			query = query.Where(ledgerentrydb.HasTransactionWith(transactionPredicates...))
		}
	}

	if input.Cursor != nil {
		query = query.Where(
			ledgerentrydb.Or(
				ledgerentrydb.CreatedAtGT(input.Cursor.Time),
				ledgerentrydb.And(
					ledgerentrydb.CreatedAt(input.Cursor.Time),
					ledgerentrydb.IDGT(input.Cursor.ID),
				),
			),
		)
	}

	query = query.Order(
		ledgerentrydb.ByCreatedAt(),
		ledgerentrydb.ByID(),
	)

	if input.Limit > 0 {
		query = query.Limit(input.Limit)
	}

	rows, err := query.All(ctx)
	if err != nil {
		return pagination.Result[ledgerhistorical.EntryData]{}, fmt.Errorf("failed to list ledger entries: %w", err)
	}

	items := lo.Map(rows, func(entity *db.LedgerEntry, _ int) ledgerhistorical.EntryData {
		dimensionIDs := fromTextArray(entity.DimensionIds)

		return ledgerhistorical.EntryData{
			ID:            entity.ID,
			Namespace:     entity.Namespace,
			Annotations:   entity.Annotations,
			CreatedAt:     entity.CreatedAt,
			AccountID:     entity.AccountID,
			AccountType:   entity.AccountType,
			DimensionIDs:  dimensionIDs,
			Amount:        entity.Amount,
			TransactionID: entity.TransactionID,
		}
	})

	var nextCursor *pagination.Cursor
	if len(items) > 0 {
		last := items[len(items)-1]
		nextCursor = lo.ToPtr(pagination.NewCursor(last.CreatedAt, last.ID))
	}

	if input.Expand.Dimensions {
		allDimensionIDs := lo.FlatMap(items, func(item ledgerhistorical.EntryData, _ int) []string {
			return item.DimensionIDs
		})
		dimensionIDs := lo.Uniq(allDimensionIDs)

		dimensionsByNamespacedID := map[string]*db.LedgerDimension{}
		if len(dimensionIDs) > 0 {
			dimensions, err := r.db.LedgerDimension.Query().
				Where(ledgerdimensiondb.IDIn(dimensionIDs...)).
				All(ctx)
			if err != nil {
				return pagination.Result[ledgerhistorical.EntryData]{}, fmt.Errorf("failed to list dimensions: %w", err)
			}

			for _, dimension := range dimensions {
				key := dimension.Namespace + ":" + dimension.ID
				dimensionsByNamespacedID[key] = dimension
			}
		}

		for i := range items {
			items[i].DimensionsExpanded = map[ledger.DimensionKey]*ledgeraccount.DimensionData{}

			for _, dimensionID := range items[i].DimensionIDs {
				key := items[i].Namespace + ":" + dimensionID
				dimension, ok := dimensionsByNamespacedID[key]
				if !ok {
					continue
				}

				dKey := ledger.DimensionKey(dimension.DimensionKey)

				items[i].DimensionsExpanded[dKey] = &ledgeraccount.DimensionData{
					ID: models.NamespacedID{
						Namespace: dimension.Namespace,
						ID:        dimension.ID,
					},
					Annotations: dimension.Annotations,
					ManagedModel: models.ManagedModel{
						CreatedAt: dimension.CreatedAt,
						UpdatedAt: dimension.UpdatedAt,
						DeletedAt: dimension.DeletedAt,
					},
					DimensionKey:          dKey,
					DimensionValue:        dimension.DimensionValue,
					DimensionDisplayValue: dimension.DimensionDisplayValue,
				}
			}
		}
	}

	return pagination.Result[ledgerhistorical.EntryData]{
		Items:      items,
		NextCursor: nextCursor,
	}, nil
}

func isInvalidDimensionReferenceError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "ledger_entries_dimension_ids_fk") ||
		(strings.Contains(msg, "SQLSTATE 23503") && strings.Contains(msg, "ledger entry references non-existent dimension id"))
}

func toTextArray(value []string) (pgtype.TextArray, error) {
	var dimensionIDs pgtype.TextArray
	if err := dimensionIDs.Set(value); err != nil {
		return pgtype.TextArray{}, err
	}

	return dimensionIDs, nil
}

func fromTextArray(value pgtype.TextArray) []string {
	var dimensionIDs []string
	if err := value.AssignTo(&dimensionIDs); err != nil {
		return nil
	}

	return dimensionIDs
}
