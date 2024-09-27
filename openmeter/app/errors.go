package app

import (
	"errors"
	"fmt"

	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	"github.com/openmeterio/openmeter/openmeter/customer"
)

var _ error = (*AppNotFoundError)(nil)

type AppNotFoundError struct {
	appentity.AppID
}

func (e AppNotFoundError) Error() string {
	return fmt.Sprintf("app with id %s not found in %s namespace", e.ID, e.Namespace)
}

var _ error = (*CustomerPreConditionError)(nil)

type CustomerPreConditionError struct {
	appentity.AppID
	AppType    appentity.AppType
	CustomerID customer.CustomerID
	Condition  string
}

func (e CustomerPreConditionError) Validate() error {
	if e.AppID.ID == "" {
		return errors.New("app id is required")
	}

	if e.AppID.Namespace == "" {
		return errors.New("app namespace is required")
	}

	if e.AppType == "" {
		return errors.New("app type is required")
	}

	if e.CustomerID.ID == "" {
		return errors.New("customer id is required")
	}

	if e.Condition == "" {
		return errors.New("condition is required")
	}

	return nil
}

func (e CustomerPreConditionError) Error() string {
	return fmt.Sprintf("customer with id %s does not meet condition %s for %s app type with id %s in namespace %s", e.CustomerID.ID, e.Condition, e.AppType, e.AppID.ID, e.AppID.Namespace)
}

var _ error = (*MarketplaceListingNotFoundError)(nil)

type MarketplaceListingNotFoundError struct {
	appentity.MarketplaceListingID
}

func (e MarketplaceListingNotFoundError) Error() string {
	return fmt.Sprintf("listing with type %s not found", e.Type)
}

type genericError struct {
	Err error
}

var _ error = (*ValidationError)(nil)

type ValidationError genericError

func (e ValidationError) Error() string {
	return e.Err.Error()
}

func (e ValidationError) Unwrap() error {
	return e.Err
}
