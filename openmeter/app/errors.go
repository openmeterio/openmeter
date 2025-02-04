package app

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/customer"
)

// AppNotFoundError
var _ error = (*AppNotFoundError)(nil)

type AppNotFoundError struct {
	AppID
}

func (e AppNotFoundError) Error() string {
	return fmt.Sprintf("app with id %s not found in %s namespace", e.ID, e.Namespace)
}

// AppDefaultNotFoundError
var _ error = (*AppDefaultNotFoundError)(nil)

type AppDefaultNotFoundError struct {
	Namespace string
	Type      AppType
}

func (e AppDefaultNotFoundError) Error() string {
	return fmt.Sprintf("there is no default app for %s type in %s namespace", e.Type, e.Namespace)
}

// AppConflictError
var _ error = (*AppConflictError)(nil)

type AppConflictError struct {
	Namespace string
	Conflict  string
}

func (e AppConflictError) Validate() error {
	if e.Namespace == "" {
		return errors.New("namespace is required")
	}

	if e.Conflict == "" {
		return errors.New("conflict reason is required")
	}

	return nil
}

func (e AppConflictError) Error() string {
	return fmt.Sprintf("app conflict: %s in namespace %s", e.Conflict, e.Namespace)
}

// AppProviderAuthenticationError
var _ error = (*AppProviderAuthenticationError)(nil)

type AppProviderAuthenticationError struct {
	AppID         *AppID
	Namespace     string
	ProviderError error
}

func (e AppProviderAuthenticationError) Error() string {
	if e.AppID == nil {
		return fmt.Sprintf("provider authentication error for app in %s namespace: %s", e.Namespace, e.ProviderError)
	}

	return fmt.Sprintf("provider authentication error for app %s: %s", e.AppID.ID, e.ProviderError)
}

// AppProviderError
var _ error = (*AppProviderError)(nil)

type AppProviderError struct {
	AppID         *AppID
	Namespace     string
	ProviderError error
}

func (e AppProviderError) Error() string {
	if e.AppID == nil {
		return fmt.Sprintf("provider error for app in %s namespace: %s", e.Namespace, e.ProviderError)
	}

	return fmt.Sprintf("provider error for app %s: %s", e.AppID.ID, e.ProviderError)
}

// AppProviderPreConditionError
var _ error = (*AppProviderPreConditionError)(nil)

type AppProviderPreConditionError struct {
	AppID     AppID
	Condition string
}

func (e AppProviderPreConditionError) Validate() error {
	if err := e.AppID.Validate(); err != nil {
		return fmt.Errorf("error validating app id: %w", err)
	}

	if e.Condition == "" {
		return errors.New("condition is required")
	}

	return nil
}

func (e AppProviderPreConditionError) Error() string {
	return fmt.Sprintf("app does not meet condition for %s: %s", e.AppID.ID, e.Condition)
}

// CustomerPreConditionError
var _ error = (*AppCustomerPreConditionError)(nil)

type AppCustomerPreConditionError struct {
	AppID
	AppType    AppType
	CustomerID customer.CustomerID
	Condition  string
}

func (e AppCustomerPreConditionError) Validate() error {
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

func (e AppCustomerPreConditionError) Error() string {
	return fmt.Sprintf("customer with id %s does not meet condition %s for %s app type with id %s in namespace %s", e.CustomerID.ID, e.Condition, e.AppType, e.AppID.ID, e.AppID.Namespace)
}

// MarketplaceListingNotFoundError
var _ error = (*MarketplaceListingNotFoundError)(nil)

type MarketplaceListingNotFoundError struct {
	MarketplaceListingID
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
