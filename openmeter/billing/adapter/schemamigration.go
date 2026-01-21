package billingadapter

import (
	"context"

	entsql "entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/customer"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billinginvoice"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type customerMigrationStatus struct {
	shouldMigrate  bool
	minSchemaLevel int
}

func (a *adapter) shouldInvoicesBeMigrated(ctx context.Context, customerID customer.CustomerID) (customerMigrationStatus, error) {
	res, err := entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (customerMigrationStatus, error) {
		schemaLevel, err := tx.GetInvoiceWriteSchemaLevel(ctx)
		if err != nil {
			return customerMigrationStatus{}, err
		}

		minInvoice, err := tx.db.BillingInvoice.Query().
			Where(billinginvoice.Namespace(customerID.Namespace)).
			Where(billinginvoice.CustomerID(customerID.ID)).
			Order(billinginvoice.BySchemaLevel(entsql.OrderAsc())).
			Select(billinginvoice.FieldSchemaLevel).
			First(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				// No invoices for this customer -> nothing to migrate.
				return customerMigrationStatus{
					shouldMigrate:  false,
					minSchemaLevel: schemaLevel,
				}, nil
			}

			return customerMigrationStatus{}, err
		}

		return customerMigrationStatus{
			shouldMigrate:  minInvoice.SchemaLevel < schemaLevel,
			minSchemaLevel: minInvoice.SchemaLevel,
		}, nil
	})
	if err != nil {
		return customerMigrationStatus{}, err
	}

	return res, nil
}

func (a *adapter) migrateCustomerInvoices(ctx context.Context, customerID customer.CustomerID, minLevel int) error {
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		if minLevel == 1 {
			err := tx.migrateSchemaLevel1(ctx, customerID)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (a *adapter) migrateSchemaLevel1(ctx context.Context, customerID customer.CustomerID) error {
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		// Schema level 1 -> 2 migration is implemented as a DB function (see migrations).
		rows, err := tx.db.QueryContext(ctx, `SELECT om_func_migrate_customer_invoices_to_schema_level_2($1)`, customerID.ID)
		if err != nil {
			return err
		}
		defer rows.Close()

		// The function returns the number of invoices updated (schema_level 1 -> 2).
		var updatedInvoiceCount int64
		if rows.Next() {
			if err := rows.Scan(&updatedInvoiceCount); err != nil {
				return err
			}
		}
		if err := rows.Err(); err != nil {
			return err
		}

		return nil
	})
}
