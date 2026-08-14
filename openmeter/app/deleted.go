package app

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/customer"
)

// DeletedApp provides the common operations for an app whose provider data is no longer available.
type DeletedApp struct {
	appID AppID
}

var _ AppOperations = DeletedApp{}

func NewDeletedApp(appID AppID) DeletedApp {
	return DeletedApp{appID: appID}
}

func (a DeletedApp) GetID() AppID {
	return a.appID
}

func (a DeletedApp) UpdateAppConfig(context.Context, AppConfigUpdate) error {
	return NewAppDeletedError(a.appID)
}

func (a DeletedApp) ValidateCapabilities(...CapabilityType) error {
	return NewAppDeletedError(a.appID)
}

func (a DeletedApp) ValidateCustomer(context.Context, *customer.Customer, []CapabilityType) error {
	return NewAppDeletedError(a.appID)
}

func (a DeletedApp) GetCustomerData(context.Context, GetAppInstanceCustomerDataInput) (CustomerData, error) {
	return nil, NewAppDeletedError(a.appID)
}

func (a DeletedApp) UpsertCustomerData(context.Context, UpsertAppInstanceCustomerDataInput) error {
	return NewAppDeletedError(a.appID)
}

func (a DeletedApp) DeleteCustomerData(context.Context, DeleteAppInstanceCustomerDataInput) error {
	return NewAppDeletedError(a.appID)
}
