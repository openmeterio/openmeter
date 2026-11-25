package billingadapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	entsql "entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/billing"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/billingcustomerlock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

var _ billing.CustomerSynchronizationAdapter = (*adapter)(nil)

func (a *adapter) LockCustomerForUpdate(ctx context.Context, input billing.LockCustomerForUpdateAdapterInput) error {
	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		// Lock the row before changing invoices, and keep it locked until the billing
		// transaction commits or rolls back.
		lockRow := func() error {
			_, err := tx.db.BillingCustomerLock.Query().
				Where(billingcustomerlock.CustomerID(input.ID)).
				Where(billingcustomerlock.Namespace(input.Namespace)).
				ForUpdate().
				First(ctx)

			return err
		}

		// Try the lock first: when the row exists, every caller queues on the row lock and
		// gets it in arrival order.
		err := lockRow()
		if err == nil {
			return nil
		}

		if !entdb.IsNotFound(err) {
			return fmt.Errorf("locking customer: %w", err)
		}

		// Create the row in the caller's billing transaction, without another connection.
		// A competing uncommitted insert blocks here until it commits or rolls back.
		err = tx.db.BillingCustomerLock.Create().
			SetNamespace(input.Namespace).
			SetCustomerID(input.ID).
			OnConflict(entsql.DoNothing()).
			Exec(ctx)
		// Ent returns sql.ErrNoRows when ON CONFLICT DO NOTHING skips the insert.
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("creating customer lock: %w", err)
		}

		// Lock with a new statement so it can see a row another transaction just committed
		// (with READ COMMITTED isolation).
		if err := lockRow(); err != nil {
			return fmt.Errorf("locking customer after creating its lock: %w", err)
		}

		return nil
	})
}
