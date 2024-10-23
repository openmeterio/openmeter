package adapter

import (
	"context"
	"fmt"

	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
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
				return nil, fmt.Errorf("failed to upsert app customer: %w", err)
			}

			return nil, nil
		},
	)

	return err
}
