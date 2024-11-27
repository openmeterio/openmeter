package app

import (
	"fmt"

	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type ListCustomerDataInput struct {
	pagination.Page
	CustomerID customerentity.CustomerID
	Type       *appentitybase.AppType
}

func (a ListCustomerDataInput) Validate() error {
	if err := a.Page.Validate(); err != nil {
		return err
	}

	if err := a.CustomerID.Validate(); err != nil {
		return err
	}

	if a.Type != nil {
		if *a.Type == "" {
			return ValidationError{
				Err: fmt.Errorf("app type cannot be empty"),
			}
		}
	}

	return nil
}

type UpsertCustomerDataInput struct {
	AppID      appentitybase.AppID
	CustomerID customerentity.CustomerID
	Data       appentity.CustomerData
}

func (a UpsertCustomerDataInput) Validate() error {
	if err := a.AppID.Validate(); err != nil {
		return err
	}

	if err := a.CustomerID.Validate(); err != nil {
		return err
	}

	if a.AppID.Namespace != a.CustomerID.Namespace {
		return ValidationError{
			Err: fmt.Errorf("app ID namespace %s does not match customer ID namespace %s", a.AppID.Namespace, a.CustomerID.Namespace),
		}
	}

	if a.Data == nil {
		return ValidationError{
			Err: fmt.Errorf("customer data cannot be nil"),
		}
	}

	if err := a.Data.Validate(); err != nil {
		return ValidationError{
			Err: fmt.Errorf("error validating customer data: %w", err),
		}
	}

	return nil
}

type DeleteCustomerDataInput struct {
	// If AppID is nil, the customer data will be deleted for all apps
	AppID      *appentitybase.AppID
	CustomerID customerentity.CustomerID
}

func (a DeleteCustomerDataInput) Validate() error {
	if a.AppID != nil {
		if err := a.AppID.Validate(); err != nil {
			return err
		}
	}

	if err := a.CustomerID.Validate(); err != nil {
		return err
	}

	if a.AppID.Namespace != a.CustomerID.Namespace {
		return ValidationError{
			Err: fmt.Errorf("app ID namespace %s does not match customer ID namespace %s", a.AppID.Namespace, a.CustomerID.Namespace),
		}
	}

	return nil
}
