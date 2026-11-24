package billingadapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	entsql "entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingcustomerlock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var _ billing.CustomerSynchronizationAdapter = (*adapter)(nil)

func (a *adapter) LockCustomerForUpdate(ctx context.Context, input billing.LockCustomerForUpdateAdapterInput) error {
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		// Create the lock row in the caller's billing transaction, without another connection.
		// A competing uncommitted insert blocks here until it commits or rolls back;
		// an existing committed row is skipped and must still be locked below.
		err := tx.db.BillingCustomerLock.Create().
			SetNamespace(input.Namespace).
			SetCustomerID(input.ID).
			OnConflict(entsql.DoNothing()).
			Exec(ctx)
		// Ent returns sql.ErrNoRows when ON CONFLICT DO NOTHING skips the insert.
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("creating customer lock: %w", err)
		}

		// Keep this query separate from the insert so it can see a row another transaction
		// just committed (with READ COMMITTED isolation). Lock the row before changing
		// invoices, and keep it locked until the billing transaction commits or rolls back.
		_, err = tx.db.BillingCustomerLock.Query().
			Where(billingcustomerlock.CustomerID(input.ID)).
			Where(billingcustomerlock.Namespace(input.Namespace)).
			ForUpdate().
			First(ctx)
		if err != nil {
			return err
		}

		return nil
	})
}
