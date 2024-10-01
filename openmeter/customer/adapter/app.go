package adapter

import (
	"context"
	"fmt"

	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
)

// UpsertAppCustomer upserts an app customer.
func (a adapter) UpsertAppCustomer(ctx context.Context, input customerentity.UpsertAppCustomerInput) error {
	err := a.db.AppCustomer.
		Create().
		SetNamespace(input.AppID.Namespace).
		SetAppID(input.AppID.ID).
		SetCustomerID(input.CustomerID).
		// Upsert
		OnConflict().
		DoNothing().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to upsert app customer: %w", err)
	}

	return nil
}
