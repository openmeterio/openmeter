package billing

import (
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/event/metadata"
	"github.com/openmeterio/openmeter/pkg/models"
)

type CollectCustomerInvoicesEvent struct {
	Namespace  string    `json:"namespace"`
	CustomerID string    `json:"customer_id"`
	AsOf       time.Time `json:"as_of"`
}

func (e CollectCustomerInvoicesEvent) EventName() string {
	return metadata.GetEventName(metadata.EventType{
		Subsystem: EventSubsystem,
		Name:      "invoice.collect",
		Version:   "v1",
	})
}

func (e CollectCustomerInvoicesEvent) EventMetadata() metadata.EventMetadata {
	return metadata.EventMetadata{
		Source:  metadata.ComposeResourcePath(e.Namespace, metadata.EntityCustomer, e.CustomerID),
		Subject: metadata.ComposeResourcePath(e.Namespace, metadata.EntityCustomer, e.CustomerID),
	}
}

func (e CollectCustomerInvoicesEvent) Validate() error {
	var errs []error

	if e.Namespace == "" {
		errs = append(errs, fmt.Errorf("namespace cannot be empty"))
	}

	if e.CustomerID == "" {
		errs = append(errs, fmt.Errorf("customer_id cannot be empty"))
	}

	if e.AsOf.IsZero() {
		errs = append(errs, fmt.Errorf("as_of cannot be zero"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
