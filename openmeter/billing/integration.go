package billing

import (
	"context"

	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
)

const (
	CapabilityInvoicing appentity.CapabilityType = "invoicing"
)

// TODO: this should be a single error but named
type ValidationErrors []error

type InvoicingIntegration struct {
	appentity.App

	SyncInvoice func(ctx context.Context, invoice Invoice) (ValidationErrors, error)
}
