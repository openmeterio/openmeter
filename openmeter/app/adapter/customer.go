package appadapter

import (
	"context"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	appcustomerdb "github.com/openmeterio/openmeter/openmeter/ent/db/appcustomer"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ app.AppAdapter = (*adapter)(nil)

// ListCustomerData lists app customer data
func (a *adapter) ListCustomerData(ctx context.Context, input app.ListCustomerInput) (pagination.Result[app.CustomerApp], error) {
	if err := input.Validate(); err != nil {
		return pagination.Result[app.CustomerApp]{}, models.NewGenericValidationError(
			fmt.Errorf("error listing customer data: %w", err),
		)
	}

	listInput := app.ListAppInput{
		Page:       input.Page,
		Namespace:  input.CustomerID.Namespace,
		CustomerID: &input.CustomerID,
		Type:       input.Type,
	}

	if input.AppID != nil {
		listInput.AppIDs = []app.AppID{*input.AppID}
	}

	apps, err := a.ListApps(ctx, listInput)
	if err != nil {
		return pagination.Result[app.CustomerApp]{}, fmt.Errorf("failed to list apps: %w", err)
	}

	response := pagination.Result[app.CustomerApp]{
		Page:       input.Page,
		TotalCount: apps.TotalCount,
		Items:      make([]app.CustomerApp, 0, len(apps.Items)),
	}

	for _, customerApp := range apps.Items {
		customerData, err := customerApp.GetCustomerData(ctx, app.GetAppInstanceCustomerDataInput{
			CustomerID: input.CustomerID,
		})
		if err != nil {
			return pagination.Result[app.CustomerApp]{}, fmt.Errorf("failed to get customer data for app %s: %w", customerApp.GetID().ID, err)
		}

		response.Items = append(response.Items, app.CustomerApp{
			App:          customerApp,
			CustomerData: customerData,
		})
	}

	return response, nil
}

// EnsureCustomer upserts app customer relationship:
// If the app or customer does not exist, an error is returned
// If the app customer relationship already exists, nothing is done
// If the app customer relationship is deleted, it is restored
func (a *adapter) EnsureCustomer(ctx context.Context, input app.EnsureCustomerInput) error {
	return transaction.RunWithNoValue(ctx, a, func(ctx context.Context) error {
		if err := input.Validate(); err != nil {
			return models.NewGenericValidationError(
				err,
			)
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
					SetNillableDeletedAt(nil).
					// Upsert
					OnConflictColumns(
						appcustomerdb.FieldNamespace,
						appcustomerdb.FieldAppID,
						appcustomerdb.FieldCustomerID,
					).
					UpdateDeletedAt().
					Exec(ctx)
				if err != nil {
					// TODO: differentiate between app or customer not found
					// When the constraint error is returned, it means that the app or customer does not exist.
					if db.IsConstraintError(err) {
						return nil, app.NewAppNotFoundError(input.AppID)
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
	})
}

// DeleteCustomer deletes app customer
func (a *adapter) DeleteCustomer(ctx context.Context, input app.DeleteCustomerInput) error {
	return transaction.RunWithNoValue(ctx, a, func(ctx context.Context) error {
		if err := input.Validate(); err != nil {
			return models.NewGenericValidationError(
				fmt.Errorf("error delete customer: %w", err),
			)
		}

		// Determine namespace
		var namespace string

		if input.AppID != nil {
			namespace = input.AppID.Namespace
		}

		if input.CustomerID != nil {
			namespace = input.CustomerID.Namespace
		}

		if namespace == "" {
			return models.NewGenericValidationError(
				fmt.Errorf("error delete customer: namespace is empty"),
			)
		}

		_, err := entutils.TransactingRepo(ctx, a, func(ctx context.Context, repo *adapter) (any, error) {
			// Delete app customer
			query := repo.db.AppCustomer.
				Update().
				SetDeletedAt(time.Now()).
				Where(
					appcustomerdb.Namespace(namespace),
				)

			if input.AppID != nil {
				query = query.Where(appcustomerdb.AppID(input.AppID.ID))
			}

			if input.CustomerID != nil {
				query = query.Where(appcustomerdb.CustomerID(input.CustomerID.ID))
			}

			_, err := query.Save(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to delete app customer: %w", err)
			}

			return nil, nil
		})
		return err
	})
}
