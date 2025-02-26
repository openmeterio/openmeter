package app

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/models"
)

// AppNotFoundError
func NewAppNotFoundError(appID AppID) *AppNotFoundError {
	return &AppNotFoundError{
		err: models.NewGenericNotFoundError(
			fmt.Errorf("app with id %s not found in %s namespace", appID.ID, appID.Namespace),
		),
	}
}

var _ models.GenericError = AppNotFoundError{}

type AppNotFoundError struct {
	err error
}

func (e AppNotFoundError) Error() string {
	return e.err.Error()
}

func (e AppNotFoundError) Unwrap() error {
	return e.err
}

// IsAppNotFoundError returns true if the error is a AppNotFoundError.
func IsAppNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	var e *AppNotFoundError

	return errors.As(err, &e)
}

// AppDefaultNotFoundError
func NewAppDefaultNotFoundError(appType AppType, namespace string) *AppDefaultNotFoundError {
	return &AppDefaultNotFoundError{
		err: models.NewGenericNotFoundError(
			fmt.Errorf("there is no default app for %s type in %s namespace", appType, namespace),
		),
	}
}

var _ models.GenericError = AppDefaultNotFoundError{}

type AppDefaultNotFoundError struct {
	err error
}

func (e AppDefaultNotFoundError) Error() string {
	return e.err.Error()
}

func (e AppDefaultNotFoundError) Unwrap() error {
	return e.err
}

func IsAppDefaultNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	var e *AppDefaultNotFoundError

	return errors.As(err, &e)
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
