package appcustominvoicing

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ app.CustomerData = (*CustomerData)(nil)

type CustomerData struct {
	Metadata models.Metadata `json:"metadata,omitempty"`
}

func (c CustomerData) Validate() error {
	return nil
}

// Customer Specific App Data Handling

func (a App) GetCustomerData(ctx context.Context, input app.GetAppInstanceCustomerDataInput) (app.CustomerData, error) {
	return a.customInvoicingService.GetCustomerData(ctx, GetAppCustomerDataInput{
		Namespace:  a.Namespace,
		AppID:      a.ID,
		CustomerID: input.CustomerID.ID,
	})
}

func (a App) UpsertCustomerData(ctx context.Context, input app.UpsertAppInstanceCustomerDataInput) error {
	data, ok := input.Data.(CustomerData)
	if !ok {
		return fmt.Errorf("invalid customer data: %v", input.Data)
	}

	return a.customInvoicingService.UpsertCustomerData(ctx, UpsertCustomerDataInput{
		CustomerDataID: CustomerDataID{
			Namespace:  a.Namespace,
			AppID:      a.ID,
			CustomerID: input.CustomerID.ID,
		},
		Data: data,
	})
}

func (a App) DeleteCustomerData(ctx context.Context, input app.DeleteAppInstanceCustomerDataInput) error {
	return a.customInvoicingService.DeleteCustomerData(ctx, DeleteAppCustomerDataInput{
		Namespace:  a.Namespace,
		AppID:      a.ID,
		CustomerID: input.CustomerID.ID,
	})
}

// Service types

type UpsertCustomerDataInput struct {
	CustomerDataID
	Data CustomerData
}

func (i UpsertCustomerDataInput) Validate() error {
	if err := i.CustomerDataID.Validate(); err != nil {
		return err
	}

	if err := i.Data.Validate(); err != nil {
		return err
	}

	return nil
}

type CustomerDataID struct {
	Namespace  string
	AppID      string
	CustomerID string
}

func (i CustomerDataID) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.CustomerID == "" {
		return errors.New("customer id is required")
	}

	if i.AppID == "" {
		return errors.New("app id is required")
	}

	return nil
}

type (
	GetAppCustomerDataInput    = CustomerDataID
	DeleteAppCustomerDataInput = CustomerDataID
)
