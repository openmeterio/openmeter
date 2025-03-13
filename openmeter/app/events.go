package app

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)

const (
	EventSubsystemAppCustomer = "app_customer"
)

type PaymentSetupAppData interface {
	Validate() error
}

type CustomerPaymentSetupResult struct {
	Metadata map[string]string `json:"metadata"`
	// AppData is the app specific data for the payment setup (the root object's App specifies the app type)
	AppData PaymentSetupAppData `json:"appData"`
}

func (r CustomerPaymentSetupResult) Validate() error {
	if err := r.AppData.Validate(); err != nil {
		return fmt.Errorf("app data: %w", err)
	}

	return nil
}

type CustomerPaymentSetupSucceededEvent struct {
	App          AppBase                    `json:"app"`
	Customer     customer.CustomerID        `json:"customer"`
	PaymentSetup CustomerPaymentSetupResult `json:"paymentSetup"`
}

var (
	_ marshaler.Event = CustomerPaymentSetupSucceededEvent{}

	appCustomerDefaultPaymentMethodChangedEventName = metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystemAppCustomer,
		Name:      "payment_setup_succeeded",
		Version:   "v1",
	})
)

func (e CustomerPaymentSetupSucceededEvent) Validate() error {
	if err := e.App.Validate(); err != nil {
		return fmt.Errorf("app: %w", err)
	}

	if err := e.Customer.Validate(); err != nil {
		return fmt.Errorf("customer: %w", err)
	}

	if err := e.PaymentSetup.Validate(); err != nil {
		return fmt.Errorf("payment setup: %w", err)
	}

	return nil
}

func (e CustomerPaymentSetupSucceededEvent) EventName() string {
	return appCustomerDefaultPaymentMethodChangedEventName
}

func (e CustomerPaymentSetupSucceededEvent) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		Source:  metadata.ComposeResourcePath(e.App.Namespace, metadata.EntityApp, e.App.ID),
		Subject: metadata.ComposeResourcePath(e.Customer.Namespace, metadata.EntityCustomer, e.Customer.ID),
	}
}
