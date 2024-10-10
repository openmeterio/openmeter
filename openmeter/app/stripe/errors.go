package appstripe

import (
	"errors"
	"fmt"

	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
)

// AppNotFoundError
var _ error = (*AppNotFoundError)(nil)

type AppNotFoundError struct {
	appentitybase.AppID
}

func (e AppNotFoundError) Error() string {
	return fmt.Sprintf("app with id %s not found in %s namespace", e.ID, e.Namespace)
}

// WebhookAppNotFoundError
var _ error = (*WebhookAppNotFoundError)(nil)

type WebhookAppNotFoundError struct {
	AppID string
}

func (e WebhookAppNotFoundError) Error() string {
	return fmt.Sprintf("app with id %s not found", e.AppID)
}

// StripeCustomerPreConditionError
var _ error = (*StripeCustomerPreConditionError)(nil)

type StripeCustomerPreConditionError struct {
	appentitybase.AppID
	AppType          appentitybase.AppType
	StripeCustomerID string
	Condition        string
}

func (e StripeCustomerPreConditionError) Validate() error {
	if e.AppID.ID == "" {
		return errors.New("app id is required")
	}

	if e.AppID.Namespace == "" {
		return errors.New("app namespace is required")
	}

	if e.AppType == "" {
		return errors.New("app type is required")
	}

	if e.StripeCustomerID == "" {
		return errors.New("stripe customer id is required")
	}

	if e.Condition == "" {
		return errors.New("condition is required")
	}

	return nil
}

func (e StripeCustomerPreConditionError) Error() string {
	return fmt.Sprintf("stripe customer with id %s does not meet condition %s for %s app type with id %s in namespace %s", e.StripeCustomerID, e.Condition, e.AppType, e.AppID.ID, e.AppID.Namespace)
}

// ValidationError
var _ error = (*ValidationError)(nil)

type ValidationError genericError

func (e ValidationError) Error() string {
	return e.Err.Error()
}

func (e ValidationError) Unwrap() error {
	return e.Err
}

// genericError
type genericError struct {
	Err error
}
