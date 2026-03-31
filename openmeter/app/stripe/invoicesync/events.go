package invoicesync

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/event/metadata"
)

const (
	EventSubsystem metadata.EventSubsystem = "app.stripe.invoicesync"
)

// ExecuteSyncPlanEvent triggers the execution of the next operation in a sync plan.
type ExecuteSyncPlanEvent struct {
	PlanID     string `json:"plan_id"`
	InvoiceID  string `json:"invoice_id"`
	Namespace  string `json:"namespace"`
	CustomerID string `json:"customer_id"`
}

func (e ExecuteSyncPlanEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "sync_plan.execute",
		Version:   "v1",
	})
}

func (e ExecuteSyncPlanEvent) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		Source:  metadata.ComposeResourcePath(e.Namespace, metadata.EntityInvoice, e.InvoiceID),
		Subject: metadata.ComposeResourcePath(e.Namespace, "stripe", "sync_plan", e.PlanID),
	}
}

func (e ExecuteSyncPlanEvent) Validate() error {
	var errs []error

	if e.PlanID == "" {
		errs = append(errs, fmt.Errorf("plan_id is required"))
	}

	if e.InvoiceID == "" {
		errs = append(errs, fmt.Errorf("invoice_id is required"))
	}

	if e.Namespace == "" {
		errs = append(errs, fmt.Errorf("namespace is required"))
	}

	if e.CustomerID == "" {
		errs = append(errs, fmt.Errorf("customer_id is required"))
	}

	return errors.Join(errs...)
}
