package appcustomeradapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/appcustomer"
	appcustomerentity "github.com/openmeterio/openmeter/openmeter/appcustomer/entity"
)

var _ appcustomer.Adapter = (*adapter)(nil)

// UpsertAppCustomer upserts an app customer.
func (a adapter) UpsertAppCustomer(ctx context.Context, input appcustomerentity.UpsertAppCustomerInput) error {
	client := a.client()

	err := client.AppCustomer.
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
