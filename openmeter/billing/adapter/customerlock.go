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

func (a *adapter) UpsertCustomerOverride(ctx context.Context, input billing.UpsertCustomerOverrideAdapterInput) error {
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
		if err := tx.UpsertCustomerOverride(ctx, input); err != nil {
			return err
		}

		_, err := tx.db.BillingCustomerLock.Query().
			Where(billingcustomerlock.CustomerID(input.ID)).
			Where(billingcustomerlock.Namespace(input.Namespace)).
			ForUpdate().
			First(ctx)

		return err
	})
}
