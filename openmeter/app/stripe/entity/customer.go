package appstripeentity

import (
	"errors"

	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
)

var _ appentity.CustomerData = (*CustomerData)(nil)

type CustomerData struct {
	AppID      appentitybase.AppID
	CustomerID customerentity.CustomerID

	StripeCustomerID             string
	StripeDefaultPaymentMethodID *string
}

func (d CustomerData) GetAppID() appentitybase.AppID {
	return d.AppID
}

func (d CustomerData) GetCustomerID() customerentity.CustomerID {
	return d.CustomerID
}

func (d CustomerData) Validate() error {
	if err := d.AppID.Validate(); err != nil {
		return err
	}

	if err := d.CustomerID.Validate(); err != nil {
		return err
	}

	if d.StripeCustomerID == "" {
		return errors.New("stripe customer id is required")
	}

	if d.StripeDefaultPaymentMethodID != nil && *d.StripeDefaultPaymentMethodID == "" {
		return errors.New("stripe default payment method id cannot be empty if provided")
	}

	return nil
}
