package adapter

import (
	"context"
	"fmt"

	sql "entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	dbledgerbreakagerecord "github.com/openmeterio/openmeter/openmeter/ent/db/ledgerbreakagerecord"
	dbledgertransaction "github.com/openmeterio/openmeter/openmeter/ent/db/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/breakage"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (a *adapter) CreateRecords(ctx context.Context, input breakage.CreateRecordsInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		creates := make([]*entdb.LedgerBreakageRecordCreate, 0, len(input.Records))
		for _, record := range input.Records {
			create := tx.db.LedgerBreakageRecord.Create().
				SetID(record.ID.ID).
				SetNamespace(record.ID.Namespace).
				SetKind(record.Kind).
				SetAmount(record.Amount).
				SetCustomerID(record.CustomerID.ID).
				SetCurrency(record.Currency).
				SetCreditPriority(record.CreditPriority).
				SetExpiresAt(record.ExpiresAt).
				SetSourceKind(record.SourceKind).
				SetNillableSourceTransactionGroupID(record.SourceTransactionGroupID).
				SetNillableSourceTransactionID(record.SourceTransactionID).
				SetNillableSourceEntryID(record.SourceEntryID).
				SetBreakageTransactionGroupID(record.BreakageTransactionGroupID).
				SetBreakageTransactionID(record.BreakageTransactionID).
				SetFboSubAccountID(record.FBOSubAccountID).
				SetBreakageSubAccountID(record.BreakageSubAccountID).
				SetNillablePlanID(record.PlanID).
				SetNillableReleaseID(record.ReleaseID).
				SetAnnotations(record.Annotations)

			creates = append(creates, create)
		}

		if len(creates) == 0 {
			return nil
		}

		if err := tx.db.LedgerBreakageRecord.CreateBulk(creates...).Exec(ctx); err != nil {
			return fmt.Errorf("create breakage record rows: %w", err)
		}

		return nil
	})
}

func (a *adapter) ListReleaseRecords(ctx context.Context, input breakage.ListReleasesInput) ([]breakage.Record, error) {
	if err := input.CustomerID.Validate(); err != nil {
		return nil, fmt.Errorf("customer id: %w", err)
	}

	if len(input.SourceEntryID) == 0 && len(input.SourceTransactionGroupID) == 0 {
		return nil, nil
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]breakage.Record, error) {
		releasePredicates := []predicate.LedgerBreakageRecord{
			dbledgerbreakagerecord.KindEQ(ledger.BreakageKindRelease),
		}
		var sourcePredicates []predicate.LedgerBreakageRecord
		if len(input.SourceEntryID) > 0 {
			sourcePredicates = append(sourcePredicates, dbledgerbreakagerecord.SourceEntryIDIn(input.SourceEntryID...))
		}
		if len(input.SourceTransactionGroupID) > 0 {
			sourcePredicates = append(sourcePredicates, dbledgerbreakagerecord.SourceTransactionGroupIDIn(input.SourceTransactionGroupID...))
		}
		if len(sourcePredicates) == 1 {
			releasePredicates = append(releasePredicates, sourcePredicates[0])
		} else {
			releasePredicates = append(releasePredicates, dbledgerbreakagerecord.Or(sourcePredicates...))
		}
		if len(input.ReleaseSourceKind) > 0 {
			releasePredicates = append(releasePredicates, dbledgerbreakagerecord.SourceKindIn(input.ReleaseSourceKind...))
		}

		rows, err := tx.db.LedgerBreakageRecord.Query().
			Where(
				dbledgerbreakagerecord.NamespaceEQ(input.CustomerID.Namespace),
				dbledgerbreakagerecord.CustomerIDEQ(input.CustomerID.ID),
				dbledgerbreakagerecord.DeletedAtIsNil(),
				dbledgerbreakagerecord.Or(
					dbledgerbreakagerecord.And(releasePredicates...),
					dbledgerbreakagerecord.KindEQ(ledger.BreakageKindReopen),
				),
			).
			Order(
				dbledgerbreakagerecord.BySourceEntryID(sql.OrderAsc()),
				dbledgerbreakagerecord.ByExpiresAt(sql.OrderAsc()),
				dbledgerbreakagerecord.ByID(sql.OrderAsc()),
			).
			ForUpdate().
			All(ctx)
		if err != nil {
			return nil, fmt.Errorf("list release breakage records: %w", err)
		}

		out := make([]breakage.Record, 0, len(rows))
		for _, row := range rows {
			out = append(out, mapRecordFromDB(row))
		}

		return out, nil
	})
}

func (a *adapter) ListExpiredRecords(ctx context.Context, input breakage.ListExpiredRecordsInput) ([]breakage.Record, error) {
	if err := input.CustomerID.Validate(); err != nil {
		return nil, fmt.Errorf("customer id: %w", err)
	}

	if input.Currency != nil {
		if err := input.Currency.Validate(); err != nil {
			return nil, fmt.Errorf("currency: %w", err)
		}
	}

	if input.AsOf.IsZero() {
		return nil, fmt.Errorf("as of is required")
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]breakage.Record, error) {
		predicates := []predicate.LedgerBreakageRecord{
			dbledgerbreakagerecord.NamespaceEQ(input.CustomerID.Namespace),
			dbledgerbreakagerecord.CustomerIDEQ(input.CustomerID.ID),
			dbledgerbreakagerecord.DeletedAtIsNil(),
			dbledgerbreakagerecord.ExpiresAtLTE(input.AsOf),
		}

		if input.Currency != nil {
			predicates = append(predicates, dbledgerbreakagerecord.CurrencyEQ(*input.Currency))
		}

		rows, err := tx.db.LedgerBreakageRecord.Query().
			Where(predicates...).
			Order(
				dbledgerbreakagerecord.ByExpiresAt(sql.OrderDesc()),
				dbledgerbreakagerecord.ByCreatedAt(sql.OrderDesc()),
				dbledgerbreakagerecord.ByID(sql.OrderDesc()),
			).
			All(ctx)
		if err != nil {
			return nil, fmt.Errorf("list expired breakage records: %w", err)
		}

		return mapRecords(rows), nil
	})
}

func (a *adapter) ListBreakageTransactionCursors(ctx context.Context, input breakage.ListBreakageTransactionCursorsInput) (map[string]ledger.TransactionCursor, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (map[string]ledger.TransactionCursor, error) {
		rows, err := tx.db.LedgerTransaction.Query().
			Where(
				dbledgertransaction.NamespaceEQ(input.Namespace),
				dbledgertransaction.IDIn(input.TransactionID...),
			).
			All(ctx)
		if err != nil {
			return nil, fmt.Errorf("list breakage transaction cursors: %w", err)
		}

		out := make(map[string]ledger.TransactionCursor, len(rows))
		for _, row := range rows {
			out[row.ID] = ledger.TransactionCursor{
				BookedAt:  row.BookedAt,
				CreatedAt: row.CreatedAt,
				ID: models.NamespacedID{
					Namespace: row.Namespace,
					ID:        row.ID,
				},
			}
		}

		return out, nil
	})
}

func (a *adapter) ListCandidateRecords(ctx context.Context, input breakage.ListPlansInput) ([]breakage.Record, error) {
	if err := input.CustomerID.Validate(); err != nil {
		return nil, fmt.Errorf("customer id: %w", err)
	}

	if err := input.Currency.Validate(); err != nil {
		return nil, fmt.Errorf("currency: %w", err)
	}

	if input.AsOf.IsZero() {
		return nil, fmt.Errorf("as of is required")
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]breakage.Record, error) {
		rows, err := tx.db.LedgerBreakageRecord.Query().
			Where(
				dbledgerbreakagerecord.NamespaceEQ(input.CustomerID.Namespace),
				dbledgerbreakagerecord.CustomerIDEQ(input.CustomerID.ID),
				dbledgerbreakagerecord.CurrencyEQ(input.Currency),
				dbledgerbreakagerecord.DeletedAtIsNil(),
				dbledgerbreakagerecord.ExpiresAtGT(input.AsOf),
				dbledgerbreakagerecord.KindIn(ledger.BreakageKindPlan, ledger.BreakageKindRelease, ledger.BreakageKindReopen),
			).
			Order(
				dbledgerbreakagerecord.ByCreditPriority(sql.OrderAsc()),
				dbledgerbreakagerecord.ByExpiresAt(sql.OrderAsc()),
				dbledgerbreakagerecord.ByID(sql.OrderAsc()),
			).
			ForUpdate().
			All(ctx)
		if err != nil {
			return nil, fmt.Errorf("list candidate breakage records: %w", err)
		}

		out := make([]breakage.Record, 0, len(rows))
		for _, row := range rows {
			out = append(out, mapRecordFromDB(row))
		}

		return out, nil
	})
}

func mapRecordFromDB(row *entdb.LedgerBreakageRecord) breakage.Record {
	return breakage.Record{
		ID: models.NamespacedID{
			Namespace: row.Namespace,
			ID:        row.ID,
		},
		Kind:      row.Kind,
		Amount:    row.Amount,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		DeletedAt: row.DeletedAt,
		CustomerID: customer.CustomerID{
			Namespace: row.Namespace,
			ID:        row.CustomerID,
		},
		Currency:                   row.Currency,
		CreditPriority:             row.CreditPriority,
		ExpiresAt:                  row.ExpiresAt,
		SourceKind:                 row.SourceKind,
		SourceTransactionGroupID:   row.SourceTransactionGroupID,
		SourceTransactionID:        row.SourceTransactionID,
		SourceEntryID:              row.SourceEntryID,
		BreakageTransactionGroupID: row.BreakageTransactionGroupID,
		BreakageTransactionID:      row.BreakageTransactionID,
		FBOSubAccountID:            row.FboSubAccountID,
		BreakageSubAccountID:       row.BreakageSubAccountID,
		PlanID:                     row.PlanID,
		ReleaseID:                  row.ReleaseID,
		Annotations:                row.Annotations,
	}
}

func mapRecords(rows []*entdb.LedgerBreakageRecord) []breakage.Record {
	out := make([]breakage.Record, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapRecordFromDB(row))
	}

	return out
}
