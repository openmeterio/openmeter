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
	Metadata map[string]string `json:"metadata,omitempty"`
	// Add additional fields here as needed. Keep in mind that this event is app neutral, so please create abstractions on top of app specific data if needed.
	// The consumer can always query the specific app data. (If this does not cut it on the long run, we need to have per app event types, which is an overkill)
}

func (r CustomerPaymentSetupResult) Validate() error {
	return nil
}

type CustomerPaymentSetupSucceededEvent struct {
	App      AppBase                    `json:"app"`
	Customer customer.CustomerID        `json:"customer"`
	Result   CustomerPaymentSetupResult `json:"result"`
}

var (
	_ marshaler.Event = CustomerPaymentSetupSucceededEvent{}

	appCustomerDefaultPaymentMethodChangedEventName = metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystemAppCustomer,
		Name:      "payment_setup_succeeded",
		Version:   "v2",
	})
)

func (e CustomerPaymentSetupSucceededEvent) Validate() error {
	if err := e.App.Validate(); err != nil {
		return fmt.Errorf("app: %w", err)
	}

	if err := e.Customer.Validate(); err != nil {
		return fmt.Errorf("customer: %w", err)
	}

	if err := e.Result.Validate(); err != nil {
		return fmt.Errorf("result: %w", err)
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
