package billingadapter

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoicewriteschemalevel"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var _ billing.SchemaLevelAdapter = (*adapter)(nil)

const (
	invoiceWriteSchemaLevelID      = "write_schema_level"
	defaultInvoiceWriteSchemaLevel = 1
)

func (a *adapter) GetInvoiceWriteSchemaLevel(ctx context.Context) (int, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (int, error) {
		record, err := tx.db.BillingInvoiceWriteSchemaLevel.Query().
			Where(billinginvoicewriteschemalevel.ID(invoiceWriteSchemaLevelID)).
			First(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return defaultInvoiceWriteSchemaLevel, nil
			}
			return 0, err
		}
		return record.SchemaLevel, nil
	})
}

func (a *adapter) SetInvoiceWriteSchemaLevel(ctx context.Context, level int) error {
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		return tx.db.BillingInvoiceWriteSchemaLevel.Create().
			SetID(invoiceWriteSchemaLevelID).
			SetSchemaLevel(level).
			OnConflict().
			UpdateSchemaLevel().
			Exec(ctx)
	})
}
