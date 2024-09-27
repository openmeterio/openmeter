package app

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/customer"
)

var _ error = (*AppNotFoundError)(nil)

type AppNotFoundError struct {
	AppID
}

func (e AppNotFoundError) Error() string {
	return fmt.Sprintf("app with id %s not found in %s namespace", e.ID, e.Namespace)
}

var _ error = (*CustomerPreConditionError)(nil)

type CustomerPreConditionError struct {
	AppID
	AppType        AppType
	AppRequirement Requirement
	CustomerID     customer.CustomerID
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

	if e.AppRequirement == "" {
		return errors.New("app requirement is required")
	}

	if e.CustomerID.ID == "" {
		return errors.New("customer id is required")
	}

	return nil
}

func (e CustomerPreConditionError) Error() string {
	return fmt.Sprintf("customer with id %s does not meet condition for %s app type with id %s in %s namespace: %s", e.CustomerID.ID, e.AppType, e.AppID.ID, e.AppID.Namespace, e.AppRequirement)
}

var _ error = (*MarketplaceListingNotFoundError)(nil)

type MarketplaceListingNotFoundError struct {
	MarketplaceListingID
}

func (e MarketplaceListingNotFoundError) Error() string {
	return fmt.Sprintf("listing with key %s not found", e.Key)
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
