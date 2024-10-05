package appstripeadapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/appstripe"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/appstripe/entity"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	appstripecustomerdb "github.com/openmeterio/openmeter/openmeter/ent/db/appstripecustomer"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

// UpsertStripeCustomerData upserts stripe customer data
func (a adapter) UpsertStripeCustomerData(ctx context.Context, input appstripeentity.UpsertStripeCustomerDataInput) error {
	if err := input.Validate(); err != nil {
		return appstripe.ValidationError{
			Err: fmt.Errorf("error upsert stripe customer data: %w", err),
		}
	}

	return transaction.RunWithNoValue(ctx, a, func(ctx context.Context) error {
		err := a.customerService.UpsertAppCustomer(ctx, customerentity.UpsertAppCustomerInput{
			AppID:      input.AppID,
			CustomerID: input.CustomerID,
		})
		if err != nil {
			return fmt.Errorf("failed to upsert app customer: %w", err)
		}

		err = a.db.AppStripeCustomer.
			Create().
			SetNamespace(input.AppID.Namespace).
			SetStripeAppID(input.AppID.ID).
			SetCustomerID(input.CustomerID.ID).
			SetStripeCustomerID(input.StripeCustomerID).
			// Upsert
			OnConflictColumns(appstripecustomerdb.FieldNamespace, appstripecustomerdb.FieldAppID, appstripecustomerdb.FieldCustomerID).
			UpdateStripeCustomerID().
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to upsert app stripe customer data: %w", err)
		}

		return nil
	})
}

// DeleteStripeCustomerData deletes stripe customer data
func (a adapter) DeleteStripeCustomerData(ctx context.Context, input appstripeentity.DeleteStripeCustomerDataInput) error {
	if err := input.Validate(); err != nil {
		return appstripe.ValidationError{
			Err: fmt.Errorf("error delete stripe customer data: %w", err),
		}
	}

	return transaction.RunWithNoValue(ctx, a, func(ctx context.Context) error {
		query := a.db.AppStripeCustomer.
			Delete().
			Where(
				appstripecustomerdb.Namespace(input.CustomerID.Namespace),
				appstripecustomerdb.CustomerID(input.CustomerID.ID),
			)

		if input.AppID != nil {
			query = query.Where(appstripecustomerdb.AppID(input.AppID.ID))
		}

		_, err := query.Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to delete app stripe customer data: %w", err)
		}

		return nil
	})
}
