package customerentity

import (
	"errors"
	"fmt"

	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
)

// CustomerApp represents an app installed for a customer
type CustomerApp struct {
	*appentitybase.AppID
	Type appentitybase.AppType `json:"type"`
	Data interface{}           `json:"data"`
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
	AppID      appentitybase.AppID
	CustomerID CustomerID
}

func (i UpsertAppCustomerInput) Validate() error {
	if err := i.AppID.Validate(); err != nil {
		return fmt.Errorf("error validating app id: %w", err)
	}

	if err := i.CustomerID.Validate(); err != nil {
		return fmt.Errorf("error validating customer id: %w", err)
	}

	if i.AppID.Namespace != i.CustomerID.Namespace {
		return errors.New("app namespace and customer namespace must match")
	}

	return nil
}
