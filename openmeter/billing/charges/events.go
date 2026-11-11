package charges

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/pkg/models"
)

// ChargeInputsChangedEvent reports persisted customer charge input changes,
// including scheduled purchases that have not produced ledger postings.
type ChargeInputsChangedEvent struct {
	Namespace  string `json:"namespace"`
	CustomerID string `json:"customerId"`
}

func (e ChargeInputsChangedEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{Subsystem: EventSubsystem, Name: "charges.inputsChanged", Version: "v1"})
}

func (e ChargeInputsChangedEvent) EventMetadata() metadata.EventMetadata {
	path := metadata.ComposeResourcePath(e.Namespace, metadata.EntityCustomer, e.CustomerID)
	return metadata.EventMetadata{Source: path, Subject: path}
}

func (e ChargeInputsChangedEvent) Validate() error {
	var errs []error
	if e.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if e.CustomerID == "" {
		errs = append(errs, errors.New("customer ID is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

const (
	EventSubsystem metadata.EventSubsystem = "billing"
)

type AdvanceChargesEvent struct {
	Namespace  string `json:"namespace"`
	CustomerID string `json:"customer_id"`
}

func (e AdvanceChargesEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "charges.advance",
		Version:   "v1",
	})
}

func (e AdvanceChargesEvent) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		Source:  metadata.ComposeResourcePath(e.Namespace, metadata.EntityCustomer, e.CustomerID),
		Subject: metadata.ComposeResourcePath(e.Namespace, metadata.EntityCustomer, e.CustomerID),
	}
}

func (e AdvanceChargesEvent) Validate() error {
	if e.Namespace == "" {
		return models.NewGenericValidationError(fmt.Errorf("namespace cannot be empty"))
	}

	if e.CustomerID == "" {
		return models.NewGenericValidationError(fmt.Errorf("customer_id cannot be empty"))
	}

	return nil
}
