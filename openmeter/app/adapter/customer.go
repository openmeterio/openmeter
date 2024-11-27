package appadapter

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/pagination"

	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appcustomerdb "github.com/openmeterio/openmeter/openmeter/ent/db/appcustomer"
)

var _ app.AppAdapter = (*adapter)(nil)

// ListCustomerData lists app customer data
func (a adapter) ListCustomerData(ctx context.Context, input app.ListCustomerDataInput) (pagination.PagedResponse[appentity.CustomerData], error) {
	if err := input.Validate(); err != nil {
		return pagination.PagedResponse[appentity.CustomerData]{}, app.ValidationError{
			Err: fmt.Errorf("error listing customer data: %w", err),
		}
	}

	apps, err := a.ListApps(ctx, appentity.ListAppInput{
		Page:       input.Page,
		Namespace:  input.CustomerID.Namespace,
		CustomerID: &input.CustomerID,
		Type:       input.Type,
	})
	if err != nil {
		return pagination.PagedResponse[appentity.CustomerData]{}, fmt.Errorf("failed to list apps: %w", err)
	}

	response := pagination.PagedResponse[appentity.CustomerData]{
		Page:       input.Page,
		TotalCount: apps.TotalCount,
		Items:      make([]appentity.CustomerData, 0, len(apps.Items)),
	}

	for _, app := range apps.Items {
		customerData, err := app.GetCustomerData(ctx, appentity.GetCustomerDataInput{
			CustomerID: input.CustomerID,
		})
		if err != nil {
			return pagination.PagedResponse[appentity.CustomerData]{}, fmt.Errorf("failed to get customer data for app %s: %w", app.GetID().ID, err)
		}

		response.Items = append(response.Items, customerData)
	}

	return response, nil
}

// UpsertCustomerData upserts app customer data
func (a adapter) UpsertCustomerData(ctx context.Context, input app.UpsertCustomerDataInput) error {
	if err := input.Validate(); err != nil {
		return app.ValidationError{
			Err: err,
		}
	}

	getApp, err := a.GetApp(ctx, input.AppID)
	if err != nil {
		return fmt.Errorf("failed to get app: %w", err)
	}

	_, err = entutils.TransactingRepo(
		ctx,
		a,
		func(ctx context.Context, repo *adapter) (any, error) {
			err = a.upsertCustomerData(ctx, input)

			// Upsert customer data for the app
			if err := getApp.UpsertCustomerData(ctx, appentity.UpsertCustomerDataInput{
				CustomerID: input.CustomerID,
				Data:       input.Data,
			}); err != nil {
				return nil, fmt.Errorf("failed to upsert customer data for app %s: %w", input.AppID.ID, err)
			}

			return nil, nil
		},
	)

	return err
}

// upsertCustomerData upserts app customer data without calling the app's UpsertCustomerData method
func (a adapter) upsertCustomerData(ctx context.Context, input app.UpsertCustomerDataInput) error {
	if err := input.Validate(); err != nil {
		return app.ValidationError{
			Err: err,
		}
	}

	_, err := entutils.TransactingRepo(
		ctx,
		a,
		func(ctx context.Context, repo *adapter) (any, error) {
			// Upsert customer data for the app
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
				if db.IsConstraintError(err) {
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

// DeleteCustomerData deletes app customer data
func (a adapter) DeleteCustomerData(ctx context.Context, input app.DeleteCustomerDataInput) error {
	if err := input.Validate(); err != nil {
		return app.ValidationError{
			Err: fmt.Errorf("error delete customer data: %w", err),
		}
	}

	apps, err := a.ListApps(ctx, appentity.ListAppInput{
		// TODO: implement get all apps instead of 10000
		Page: pagination.Page{
			PageNumber: 1,
			PageSize:   10000,
		},
		Namespace:  input.CustomerID.Namespace,
		CustomerID: &input.CustomerID,
	})
	if err != nil {
		return fmt.Errorf("failed to list apps: %w", err)
	}

	_, err = entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *adapter) (any, error) {
		// Delete customer data for all apps
		for _, app := range apps.Items {
			if err := app.DeleteCustomerData(ctx, appentity.DeleteCustomerDataInput{
				CustomerID: input.CustomerID,
			}); err != nil {
				return nil, fmt.Errorf("failed to delete customer data for app %s: %w", app.GetID().ID, err)
			}
		}

		// Delete app customer data
		query := repo.db.AppCustomer.
			Delete().
			Where(
				appcustomerdb.Namespace(input.CustomerID.Namespace),
				appcustomerdb.CustomerID(input.CustomerID.ID),
			)

		if input.AppID != nil {
			query = query.Where(appcustomerdb.AppID(input.AppID.ID))
		}

		_, err := query.Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to delete app customer data: %w", err)
		}

		return nil, nil
	})
	return err
}
