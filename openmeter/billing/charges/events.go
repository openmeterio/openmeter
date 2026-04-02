package charges

import (
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/pkg/models"
)

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
