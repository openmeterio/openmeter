package charges

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

type Service interface {
	ChargeService

	BillingEmulationService
}

type ChargeService interface {
	GetChargeByID(ctx context.Context, input ChargeID) (Charge, error)
	GetChargesByIDs(ctx context.Context, ns string, ids []string) (Charges, error)
	CreateCharges(ctx context.Context, input CreateChargeInput) (Charges, error)
}

type BillingEmulationService interface {
	InvoicePendingLines(ctx context.Context, input billing.InvoicePendingLinesInput) ([]billing.StandardInvoice, error)
}
