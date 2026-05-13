package adapter

import (
	"context"
	"fmt"

	sql "entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	dbledgerbreakagerecord "github.com/openmeterio/openmeter/openmeter/ent/db/ledgerbreakagerecord"
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
		BreakageTransactionGroupID: row.BreakageTransactionGroupID,
		BreakageTransactionID:      row.BreakageTransactionID,
		FBOSubAccountID:            row.FboSubAccountID,
		BreakageSubAccountID:       row.BreakageSubAccountID,
		PlanID:                     row.PlanID,
		ReleaseID:                  row.ReleaseID,
		Annotations:                row.Annotations,
	}
}
