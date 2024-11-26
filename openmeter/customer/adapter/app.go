package adapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

// UpsertAppCustomer upserts an app customer.
func (a adapter) UpsertAppCustomer(ctx context.Context, input customerentity.UpsertAppCustomerInput) error {
	_, err := entutils.TransactingRepo(
		ctx,
		a,
		func(ctx context.Context, repo *adapter) (any, error) {
			err := repo.db.AppCustomer.
				Create().
				SetNamespace(input.AppID.Namespace).
				SetAppID(input.AppID.ID).
				SetCustomerID(input.CustomerID.ID).
				// Upsert
				OnConflict().
				DoNothing().
				Exec(ctx)
			if err != nil {
				// TODO: differentiate between app or customer not found
				// When the constraint error is returned, it means that the app or customer does not exist.
				if entdb.IsConstraintError(err) {
					return nil, app.AppNotFoundError{
						AppID: input.AppID,
					}
				}

				// TODO (pmarton): This is a workaround for the issue where DoNothing() returns an error when no rows are affected.
				// See: https://github.com/ent/ent/issues/1821
				if err.Error() == "sql: no rows in result set" {
					return nil, nil
				}

				return nil, fmt.Errorf("failed to upsert app customer: %w", err)
			}

			return nil, nil
		},
	)

	return err
}
