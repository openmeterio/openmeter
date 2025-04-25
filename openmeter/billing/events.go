package billing

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/event/metadata"
)

const (
	EventSubsystem metadata.EventSubsystem = "billing"
)

type InvoiceAppBases struct {
	Tax      app.AppBase `json:"tax"`
	Payment  app.AppBase `json:"payment"`
	Invocing app.AppBase `json:"invocing"`
}

func (a InvoiceAppBases) Validate() error {
	var errs []error

	if err := a.Tax.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("tax: %w", err))
	}

	if err := a.Payment.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("payment: %w", err))
	}

	if err := a.Invocing.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("invocing: %w", err))
	}

	return errors.Join(errs...)
}

type EventInvoice struct {
	Invoice  Invoice         `json:"invoice"`
	AppBases InvoiceAppBases `json:"app_bases,omitempty"`
}

func (e EventInvoice) Validate() error {
	var errs []error

	if err := e.Invoice.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := e.AppBases.Validate(); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func NewEventInvoice(invoice Invoice) EventInvoice {
	// This causes a stack overflow
	payload := invoice.RemoveCircularReferences()

	// Remove the Apps from the payload, as they are not json unmarshallable
	// but either ways, the apps service should be used in workers to acquire
	// an up-to-date app based on the payload.Workflow.AppReferences
	payload.Workflow.Apps = nil

	appBases := InvoiceAppBases{}
	// TODO[later]: Apps are always present, so we should only use the struct without a pointer
	if invoice.Workflow.Apps != nil {
		appBases = InvoiceAppBases{
			Tax:      invoice.Workflow.Apps.Tax.GetAppBase(),
			Payment:  invoice.Workflow.Apps.Payment.GetAppBase(),
			Invocing: invoice.Workflow.Apps.Invoicing.GetAppBase(),
		}
	}

	return EventInvoice{
		Invoice:  payload,
		AppBases: appBases,
	}
}

type InvoiceCreatedEvent struct {
	EventInvoice `json:",inline"`
}

func NewInvoiceCreatedEvent(invoice Invoice) InvoiceCreatedEvent {
	return InvoiceCreatedEvent{NewEventInvoice(invoice)}
}

func (e InvoiceCreatedEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "invoice.created",
		Version:   "v2",
	})
}

func (e InvoiceCreatedEvent) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		Source:  metadata.ComposeResourcePath(e.Invoice.Namespace, metadata.EntityInvoice, e.Invoice.ID),
		Subject: metadata.ComposeResourcePath(e.Invoice.Namespace, metadata.EntityCustomer, e.Invoice.Customer.CustomerID),
	}
}

func (e InvoiceCreatedEvent) Validate() error {
	return e.EventInvoice.Validate()
}

type AdvanceInvoiceEvent struct {
	Invoice    InvoiceID `json:"invoice"`
	CustomerID string    `json:"customer_id"`
}

func (e AdvanceInvoiceEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "invoice.advance",
		Version:   "v1",
	})
}

func (e AdvanceInvoiceEvent) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		Source:  metadata.ComposeResourcePath(e.Invoice.Namespace, metadata.EntityInvoice, e.Invoice.ID),
		Subject: metadata.ComposeResourcePath(e.Invoice.Namespace, metadata.EntityCustomer, e.CustomerID),
	}
}

func (e AdvanceInvoiceEvent) Validate() error {
	if err := e.Invoice.Validate(); err != nil {
		return err
	}

	if e.CustomerID == "" {
		return fmt.Errorf("customer_id cannot be empty")
	}

	return nil
}
