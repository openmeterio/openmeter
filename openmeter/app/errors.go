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
func NewAppProviderAuthenticationError(appID *AppID, namespace string, providerError error) *AppProviderAuthenticationError {
	var err error

	if appID == nil {
		err = fmt.Errorf("provider authentication error for app in %s namespace: %w", namespace, providerError)
	} else {
		err = fmt.Errorf("provider authentication error for app %s: %w", appID.ID, providerError)
	}

	return &AppProviderAuthenticationError{
		err: models.NewGenericUnauthorizedError(err),
	}
}

var _ models.GenericError = (*AppProviderAuthenticationError)(nil)

type AppProviderAuthenticationError struct {
	err error
}

func (e AppProviderAuthenticationError) Error() string {
	return e.err.Error()
}

func (e AppProviderAuthenticationError) Unwrap() error {
	return e.err
}

func IsAppProviderAuthenticationError(err error) bool {
	if err == nil {
		return false
	}

	var e *AppProviderAuthenticationError

	return errors.As(err, &e)
}

// AppProviderError
func NewAppProviderError(appID *AppID, namespace string, providerError error) *AppProviderError {
	var err error

	if appID == nil {
		err = fmt.Errorf("provider error for app in %s namespace: %w", namespace, providerError)
	} else {
		err = fmt.Errorf("provider error for app %s: %w", appID.ID, providerError)
	}

	return &AppProviderError{
		err: models.NewGenericPreConditionFailedError(err),
	}
}

var _ models.GenericError = (*AppProviderError)(nil)

type AppProviderError struct {
	err error
}

func (e AppProviderError) Error() string {
	return e.err.Error()
}

func (e AppProviderError) Unwrap() error {
	return e.err
}

func IsAppProviderError(err error) bool {
	if err == nil {
		return false
	}

	var e *AppProviderError

	return errors.As(err, &e)
}

// AppProviderPreConditionError
var _ models.GenericError = (*AppProviderPreConditionError)(nil)

func NewAppProviderPreConditionError(appID AppID, condition string) *AppProviderPreConditionError {
	return &AppProviderPreConditionError{
		err: models.NewGenericPreConditionFailedError(
			fmt.Errorf("app does not meet condition for %s: %s", appID.ID, condition),
		),
	}
}

type AppProviderPreConditionError struct {
	err error
}

func (e AppProviderPreConditionError) Error() string {
	return e.err.Error()
}

func (e AppProviderPreConditionError) Unwrap() error {
	return e.err
}

func IsAppProviderPreConditionError(err error) bool {
	if err == nil {
		return false
	}

	var e *AppProviderPreConditionError

	return errors.As(err, &e)
}

// AppCustomerPreConditionError
func NewAppCustomerPreConditionError(appID AppID, appType AppType, customerID *customer.CustomerID, condition string) *AppCustomerPreConditionError {
	var err error

	if customerID == nil {
		err = fmt.Errorf("customer does not meet condition for %s app type with id %s in namespace %s: %s", appType, appID.ID, appID.Namespace, condition)
	} else {
		err = fmt.Errorf("customer with id %s does not meet condition %s for %s app type with id %s in namespace %s", customerID.ID, condition, appType, appID.ID, appID.Namespace)
	}

	return &AppCustomerPreConditionError{
		err: models.NewGenericPreConditionFailedError(err),
	}
}

var _ models.GenericError = (*AppCustomerPreConditionError)(nil)

type AppCustomerPreConditionError struct {
	err error
}

func (e AppCustomerPreConditionError) Error() string {
	return e.err.Error()
}

func (e AppCustomerPreConditionError) Unwrap() error {
	return e.err
}

func IsAppCustomerPreConditionError(err error) bool {
	if err == nil {
		return false
	}

	var e *AppCustomerPreConditionError

	return errors.As(err, &e)
}
