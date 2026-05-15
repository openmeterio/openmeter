package app

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type ListCustomerInput struct {
	pagination.Page
	AppID      *AppID
	CustomerID customer.CustomerID
	Type       *AppType
}

func (a ListCustomerInput) Validate() error {
	var errs []error

	if err := a.CustomerID.Validate(); err != nil {
		errs = append(errs, err)
	}

	if a.AppID != nil {
		if err := a.AppID.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	if a.Type != nil {
		if *a.Type == "" {
			errs = append(errs, models.NewGenericValidationError(
				fmt.Errorf("app type cannot be empty"),
			))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type EnsureCustomerInput struct {
	AppID      AppID
	CustomerID customer.CustomerID
}

func (a EnsureCustomerInput) Validate() error {
	var errs []error

	if err := a.AppID.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := a.CustomerID.Validate(); err != nil {
		errs = append(errs, err)
	}

	if a.AppID.Namespace != a.CustomerID.Namespace {
		errs = append(errs, models.NewGenericValidationError(fmt.Errorf("app ID namespace %s does not match customer ID namespace %s", a.AppID.Namespace, a.CustomerID.Namespace)))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type DeleteCustomerInput struct {
	AppID      *AppID
	CustomerID *customer.CustomerID
}

func (a DeleteCustomerInput) Validate() error {
	var errs []error

	if a.AppID == nil && a.CustomerID == nil {
		errs = append(errs, models.NewGenericValidationError(errors.New("app ID and customer ID cannot be nil")))
	}

	if a.AppID != nil {
		if err := a.AppID.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	if a.CustomerID != nil {
		if err := a.CustomerID.Validate(); err != nil {
			errs = append(errs, err)
		}
	}

	if a.AppID != nil && a.CustomerID != nil && a.AppID.Namespace != a.CustomerID.Namespace {
		errs = append(errs, models.NewGenericValidationError(errors.New("app and customer must be in the same namespace")))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
