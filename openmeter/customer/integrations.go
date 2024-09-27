package customer

import (
	"context"

	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
)

const (
	CapabilityCustomerManagement appentity.CapabilityType = "manageCustomers"
)

type Integration interface {
	appentity.App
	ValidateCustomer(ctx context.Context, customer Customer) error
	UpsertCustomer(ctx context.Context, customer Customer) error

	// TODO: this might be a seperate capability
	ImportCustomers(ctx context.Context) ([]CustomerImportInput, error)
}

type CustomerImportInput struct {
	AppID      appentity.AppID
	ExternalID string
	Customer   Customer
}
