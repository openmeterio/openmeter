package billingadapter

import (
	"context"
	"database/sql"

	entsql "entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingcustomerlock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var _ billing.CustomerOverrideAdapter = (*adapter)(nil)

func (a *adapter) UpsertCustomerLock(ctx context.Context, input billing.UpsertCustomerLockAdapterInput) error {
	err := a.db.BillingCustomerLock.Create().
		SetNamespace(input.Namespace).
		SetCustomerID(input.ID).
		OnConflict(
			entsql.DoNothing(),
		).
		Exec(ctx)
	if err != nil {
		// The do nothing returns no lines, so we have the record ready
		if err == sql.ErrNoRows {
			return nil
		}
	}
	return nil
}

func (a *adapter) LockCustomerForUpdate(ctx context.Context, input billing.LockCustomerForUpdateAdapterInput) error {
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		if err := tx.UpsertCustomerLock(ctx, input); err != nil {
			return err
		}

		_, err := tx.db.BillingCustomerLock.Query().
			Where(billingcustomerlock.CustomerID(input.ID)).
			Where(billingcustomerlock.Namespace(input.Namespace)).
			ForUpdate().
			First(ctx)
		if err != nil {
			return err
		}

		// Temp: until the migrations are complete
		migrationStatus, err := tx.shouldInvoicesBeMigrated(ctx, input)
		if err != nil {
			return err
		}

		if migrationStatus.shouldMigrate {
			err := tx.migrateCustomerInvoices(ctx, input, migrationStatus.minSchemaLevel)
			if err != nil {
				return err
			}
		}

		return nil
	})
}
