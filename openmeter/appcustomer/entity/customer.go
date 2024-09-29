package appcustomer

import (
	"errors"
	"fmt"

	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
)

// CustomerApp represents an app installed for a customer
type CustomerApp struct {
	*appentity.AppID
	Type appentity.AppType `json:"type"`
	Data interface{}       `json:"data"`
}

func (a CustomerApp) Validate() error {
	if a.ID == "" {
		return errors.New("app id is required")
	}

	if a.Namespace == "" {
		return errors.New("app namespace is required")
	}

	if a.Type == "" {
		return errors.New("app type is required")
	}

	return nil
}

type UpsertAppCustomerInput struct {
	AppID appentity.AppID
	// TODO: use customer.CustomerID without cyclic dependency
	CustomerID string
}

func (i UpsertAppCustomerInput) Validate() error {
	if err := i.AppID.Validate(); err != nil {
		return fmt.Errorf("error validating app id: %w", err)
	}

	// if err := i.CustomerID.Validate(); err != nil {
	// 	return fmt.Errorf("error validating customer id: %w", err)
	// }
	if i.CustomerID == "" {
		return errors.New("customer id is required")
	}

	return nil
}
