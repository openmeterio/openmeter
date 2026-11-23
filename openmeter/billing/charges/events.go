package charges

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/pkg/models"
)

// CustomerChargesChangedEvent signals that creating or patching charges may have
// changed a customer's credit balances, even before a ledger posting exists.
// A batch can change several charges. RTE uses the customer ID to invalidate
// tracked balances across currencies and feature scopes.
type CustomerChargesChangedEvent struct {
	Namespace  string `json:"namespace"`
	CustomerID string `json:"customerId"`
}

func (e CustomerChargesChangedEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{Subsystem: EventSubsystem, Name: "charges.inputsChanged", Version: "v1"})
}

func (e CustomerChargesChangedEvent) EventMetadata() metadata.EventMetadata {
	path := metadata.ComposeResourcePath(e.Namespace, metadata.EntityCustomer, e.CustomerID)
	return metadata.EventMetadata{Source: path, Subject: path}
}

func (e CustomerChargesChangedEvent) Validate() error {
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
