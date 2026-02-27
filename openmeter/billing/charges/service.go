package charges

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

type Service interface {
	ChargeService
	InvoiceService
}

type ChargeService interface {
	GetChargeByID(ctx context.Context, input ChargeID) (Charge, error)
	GetChargesByIDs(ctx context.Context, namespace string, ids []string) (Charges, error)
	CreateCharges(ctx context.Context, input CreateChargeInputs) (Charges, error)
}

// InvoiceService contains methods that are over time deprecate the current billing methods.
type InvoiceService interface {
	InvoicePendingLines(ctx context.Context, input billing.InvoicePendingLinesInput) ([]billing.StandardInvoice, error)
}
