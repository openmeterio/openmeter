package app

import (
	"errors"
	"fmt"

	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type ListCustomerInput struct {
	pagination.Page
	CustomerID customerentity.CustomerID
	Type       *appentitybase.AppType
}

func (a ListCustomerInput) Validate() error {
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

type EnsureCustomerInput struct {
	AppID      appentitybase.AppID
	CustomerID customerentity.CustomerID
}

func (a EnsureCustomerInput) Validate() error {
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

	return nil
}

type DeleteCustomerInput struct {
	AppID      *appentitybase.AppID
	CustomerID *customerentity.CustomerID
}

func (a DeleteCustomerInput) Validate() error {
	if a.AppID == nil && a.CustomerID == nil {
		return ValidationError{
			Err: fmt.Errorf("app ID and customer ID cannot be nil"),
		}
	}

	if a.AppID != nil {
		if err := a.AppID.Validate(); err != nil {
			return err
		}
	}

	if a.CustomerID != nil {
		if err := a.CustomerID.Validate(); err != nil {
			return err
		}
	}

	if a.AppID != nil && a.CustomerID != nil && a.AppID.Namespace != a.CustomerID.Namespace {
		return errors.New("app and customer must be in the same namespace")
	}

	return nil
}
