package adapter

import (
	"context"
	"time"

	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/ent/db/appcustominvoicingcustomer"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
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
		return tx.db.AppCustomInvoicingCustomer.Create().
			SetMetadata(input.Data.Metadata).
			SetCustomerID(input.CustomerDataID.CustomerID).
			SetNamespace(input.CustomerDataID.Namespace).
			SetAppID(input.CustomerDataID.AppID).
			OnConflictColumns(appcustominvoicingcustomer.FieldCustomerID, appcustominvoicingcustomer.FieldNamespace, appcustominvoicingcustomer.FieldAppID).
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
			).
			Exec(ctx)
	})
}

func mapDBToCustomerData(line *db.AppCustomInvoicingCustomer) appcustominvoicing.CustomerData {
	return appcustominvoicing.CustomerData{
		Metadata: line.Metadata,
	}
}
