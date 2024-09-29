package appcustomer

import (
	"errors"

	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
)

// CustomerApp represents an app installed for a customer
type CustomerApp struct {
	appentity.AppID
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
