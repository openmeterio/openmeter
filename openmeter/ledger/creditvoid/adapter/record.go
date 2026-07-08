package adapter

import (
	"context"
	"fmt"

	sql "entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	dbledgercreditvoidrecord "github.com/openmeterio/openmeter/openmeter/ent/db/ledgercreditvoidrecord"
	"github.com/openmeterio/openmeter/openmeter/ent/db/predicate"
	"github.com/openmeterio/openmeter/openmeter/ledger/creditvoid"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (a *adapter) CreateRecords(ctx context.Context, input creditvoid.CreateRecordsInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		creates := make([]*entdb.LedgerCreditVoidRecordCreate, 0, len(input.Records))
		for _, record := range input.Records {
			create := tx.db.LedgerCreditVoidRecord.Create().
				SetID(record.ID.ID).
				SetNamespace(record.ID.Namespace).
				SetAmount(record.Amount).
				SetCustomerID(record.CustomerID.ID).
				SetCurrency(record.Currency).
				SetVoidedAt(record.VoidedAt).
				SetSourceChargeID(record.SourceChargeID).
				SetVoidTransactionGroupID(record.VoidTransactionGroupID).
				SetVoidTransactionID(record.VoidTransactionID).
				SetFboSubAccountID(record.FBOSubAccountID).
				SetReceivableSubAccountID(record.ReceivableSubAccountID).
				SetAnnotations(record.Annotations)

			creates = append(creates, create)
		}

		if len(creates) == 0 {
			return nil
		}

		if err := tx.db.LedgerCreditVoidRecord.CreateBulk(creates...).Exec(ctx); err != nil {
			return fmt.Errorf("create credit void record rows: %w", err)
		}

		return nil
	})
}

func (a *adapter) ListRecords(ctx context.Context, input creditvoid.ListRecordsInput) ([]creditvoid.Record, error) {
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

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]creditvoid.Record, error) {
		predicates := []predicate.LedgerCreditVoidRecord{
			dbledgercreditvoidrecord.NamespaceEQ(input.CustomerID.Namespace),
			dbledgercreditvoidrecord.CustomerIDEQ(input.CustomerID.ID),
			dbledgercreditvoidrecord.DeletedAtIsNil(),
			dbledgercreditvoidrecord.VoidedAtLTE(input.AsOf),
		}

		if input.Currency != nil {
			predicates = append(predicates, dbledgercreditvoidrecord.CurrencyEQ(*input.Currency))
		}
		if routePredicate := voidRecordRoutePredicate(input.Route); routePredicate != nil {
			predicates = append(predicates, routePredicate)
		}

		rows, err := tx.db.LedgerCreditVoidRecord.Query().
			Where(predicates...).
			Order(
				dbledgercreditvoidrecord.ByVoidedAt(sql.OrderDesc()),
				dbledgercreditvoidrecord.ByCreatedAt(sql.OrderDesc()),
				dbledgercreditvoidrecord.ByID(sql.OrderDesc()),
			).
			All(ctx)
		if err != nil {
			return nil, fmt.Errorf("list credit void records: %w", err)
		}

		return mapRecords(rows), nil
	})
}

func mapRecordFromDB(row *entdb.LedgerCreditVoidRecord) creditvoid.Record {
	return creditvoid.Record{
		ID: models.NamespacedID{
			Namespace: row.Namespace,
			ID:        row.ID,
		},
		Amount:                 row.Amount,
		CreatedAt:              row.CreatedAt,
		UpdatedAt:              row.UpdatedAt,
		DeletedAt:              row.DeletedAt,
		CustomerID:             customer.CustomerID{Namespace: row.Namespace, ID: row.CustomerID},
		Currency:               row.Currency,
		VoidedAt:               row.VoidedAt,
		SourceChargeID:         row.SourceChargeID,
		VoidTransactionGroupID: row.VoidTransactionGroupID,
		VoidTransactionID:      row.VoidTransactionID,
		FBOSubAccountID:        row.FboSubAccountID,
		ReceivableSubAccountID: row.ReceivableSubAccountID,
		Annotations:            row.Annotations,
	}
}

func mapRecords(rows []*entdb.LedgerCreditVoidRecord) []creditvoid.Record {
	out := make([]creditvoid.Record, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapRecordFromDB(row))
	}

	return out
}
