package adapter

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/app"
	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	appcustominvoicingdb "github.com/openmeterio/openmeter/openmeter/ent/db/appcustominvoicing"
	"github.com/openmeterio/openmeter/openmeter/ent/db/appcustominvoicingcustomer"
	customerdb "github.com/openmeterio/openmeter/openmeter/ent/db/customer"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (a *adapter) GetCustomerData(ctx context.Context, input appcustominvoicing.GetAppCustomerDataInput) (appcustominvoicing.CustomerData, error) {
	if err := input.Validate(); err != nil {
		return appcustominvoicing.CustomerData{}, err
	}

	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (appcustominvoicing.CustomerData, error) {
		line, err := tx.db.AppCustomInvoicingCustomer.Query().
			Where(
				appcustominvoicingcustomer.CustomerID(input.CustomerID),
				appcustominvoicingcustomer.Namespace(input.Namespace),
				appcustominvoicingcustomer.AppID(input.AppID),
				appcustominvoicingcustomer.DeletedAtIsNil(),
			).
			First(ctx)
		if err != nil {
			if db.IsNotFound(err) {
				return appcustominvoicing.CustomerData{}, nil
			}

			return appcustominvoicing.CustomerData{}, err
		}

		return mapDBToCustomerData(line), nil
	})
}

func (a *adapter) UpsertCustomerData(ctx context.Context, input appcustominvoicing.UpsertCustomerDataInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		appExists, err := tx.db.AppCustomInvoicing.Query().
			Where(appcustominvoicingdb.Namespace(input.CustomerDataID.Namespace)).
			Where(appcustominvoicingdb.ID(input.CustomerDataID.AppID)).
			Where(appcustominvoicingdb.DeletedAtIsNil()).
			Exist(ctx)
		if err != nil {
			return fmt.Errorf("failed to resolve custom invoicing app: %w", err)
		}

		if !appExists {
			return app.NewAppNotFoundError(app.AppID{
				Namespace: input.CustomerDataID.Namespace,
				ID:        input.CustomerDataID.AppID,
			})
		}

		customerExists, err := tx.db.Customer.Query().
			Where(customerdb.Namespace(input.CustomerDataID.Namespace)).
			Where(customerdb.ID(input.CustomerDataID.CustomerID)).
			Where(customerdb.DeletedAtIsNil()).
			Exist(ctx)
		if err != nil {
			return fmt.Errorf("failed to resolve customer: %w", err)
		}

		if !customerExists {
			return models.NewGenericNotFoundError(
				fmt.Errorf("customer with id %s not found in %s namespace", input.CustomerDataID.CustomerID, input.CustomerDataID.Namespace),
			)
		}

		return tx.db.AppCustomInvoicingCustomer.Create().
			SetMetadata(input.Data.Metadata).
			SetCustomerID(input.CustomerDataID.CustomerID).
			SetNamespace(input.CustomerDataID.Namespace).
			SetAppID(input.CustomerDataID.AppID).
			// Upsert
			OnConflict(
				sql.ConflictColumns(
					appcustominvoicingcustomer.FieldCustomerID,
					appcustominvoicingcustomer.FieldNamespace,
					appcustominvoicingcustomer.FieldAppID,
				),
				sql.ConflictWhere(sql.IsNull(appcustominvoicingcustomer.FieldDeletedAt)),
			).
			UpdateMetadata().
			UpdateDeletedAt().
			Exec(ctx)
	})
}

func (a *adapter) DeleteCustomerData(ctx context.Context, input appcustominvoicing.DeleteAppCustomerDataInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
		return tx.db.AppCustomInvoicingCustomer.Update().
			SetDeletedAt(time.Now()).
			Where(
				appcustominvoicingcustomer.CustomerID(input.CustomerID),
				appcustominvoicingcustomer.Namespace(input.Namespace),
				appcustominvoicingcustomer.AppID(input.AppID),
				appcustominvoicingcustomer.DeletedAtIsNil(),
			).
			Exec(ctx)
	})
}

func mapDBToCustomerData(line *db.AppCustomInvoicingCustomer) appcustominvoicing.CustomerData {
	return appcustominvoicing.CustomerData{
		Metadata: line.Metadata,
	}
}
