package appstripeentity

import (
	"errors"
	"fmt"

	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
)

type CreateAppStripeInput struct {
	Namespace       string
	Name            string
	Description     string
	StripeAccountID string
	Livemode        bool
	APIKey          secretentity.SecretID
}

func (i CreateAppStripeInput) Validate() error {
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	if i.Name == "" {
		return errors.New("name is required")
	}

	if i.Description == "" {
		return errors.New("description is required")
	}

	if i.StripeAccountID == "" {
		return errors.New("stripe account id is required")
	}

	if err := i.APIKey.Validate(); err != nil {
		return fmt.Errorf("error validating api key: %w", err)
	}

	if i.APIKey.Namespace != i.Namespace {
		return errors.New("api key must be in the same namespace as the app")
	}

	return nil
}

type UpsertStripeCustomerDataInput struct {
	AppID            appentitybase.AppID
	CustomerID       customerentity.CustomerID
	StripeCustomerID string
}

func (i UpsertStripeCustomerDataInput) Validate() error {
	if err := i.AppID.Validate(); err != nil {
		return fmt.Errorf("error validating app id: %w", err)
	}

	if err := i.CustomerID.Validate(); err != nil {
		return fmt.Errorf("error validating customer id: %w", err)
	}

	if i.AppID.Namespace != i.CustomerID.Namespace {
		return errors.New("app and customer must be in the same namespace")
	}

	if i.StripeCustomerID == "" {
		return errors.New("stripe customer id is required")
	}

	return nil
}

type DeleteStripeCustomerDataInput struct {
	AppID      *appentitybase.AppID
	CustomerID customerentity.CustomerID
}

func (i DeleteStripeCustomerDataInput) Validate() error {
	if i.CustomerID.ID == "" {
		return errors.New("customer id is required")
	}

	if i.CustomerID.Namespace == "" {
		return errors.New("customer namespace is required")
	}

	if i.AppID != nil {
		if i.AppID.ID == "" {
			return errors.New("app id is required")
		}

		if i.AppID.Namespace == "" {
			return errors.New("app namespace is required")
		}

		if i.AppID.Namespace != i.CustomerID.Namespace {
			return errors.New("app and customer must be in the same namespace")
		}
	}

	return nil
}
