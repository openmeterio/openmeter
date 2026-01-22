package billingadapter

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoice"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoicewriteschemalevel"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var _ billing.SchemaLevelAdapter = (*adapter)(nil)

const (
	invoiceWriteSchemaLevelID      = "write_schema_level"
	DefaultInvoiceWriteSchemaLevel = 1
)

func (a *adapter) GetInvoiceDefaultSchemaLevel(ctx context.Context) (int, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (int, error) {
		record, err := tx.db.BillingInvoiceWriteSchemaLevel.Query().
			Where(billinginvoicewriteschemalevel.ID(invoiceWriteSchemaLevelID)).
			First(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return DefaultInvoiceWriteSchemaLevel, nil
			}
			return 0, err
		}
		return record.SchemaLevel, nil
	})
}

func (a *adapter) SetInvoiceDefaultSchemaLevel(ctx context.Context, level int) error {
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		return tx.db.BillingInvoiceWriteSchemaLevel.Create().
			SetID(invoiceWriteSchemaLevelID).
			SetSchemaLevel(level).
			OnConflictColumns(billinginvoicewriteschemalevel.FieldID).
			UpdateSchemaLevel().
			Exec(ctx)
	})
}

func (a *adapter) getSchemaLevelPerInvoice(ctx context.Context, customerID customer.CustomerID) (map[string]int, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (map[string]int, error) {
		invoices, err := tx.db.BillingInvoice.Query().
			Where(billinginvoice.Namespace(customerID.Namespace)).
			Where(billinginvoice.CustomerID(customerID.ID)).
			Select(billinginvoice.FieldID, billinginvoice.FieldSchemaLevel).
			All(ctx)
		if err != nil {
			return nil, err
		}

		out := make(map[string]int, len(invoices))
		for _, inv := range invoices {
			out[inv.ID] = inv.SchemaLevel
		}

		return out, nil
	})
}
