package usagebased

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Adapter interface {
	RealizationRunAdapter
	RealizationRunCreditAllocationAdapter
	RealizationRunInvoiceUsageAdapter
	RealizationRunPaymentAdapter
	ChargeAdapter

	entutils.TxCreator
}

type ChargeAdapter interface {
	CreateCharges(ctx context.Context, charges CreateChargesInput) ([]Charge, error)
	UpdateCharge(ctx context.Context, charge ChargeBase) (ChargeBase, error)
	DeleteCharge(ctx context.Context, charge Charge) error
	GetByIDs(ctx context.Context, input GetByIDsInput) ([]Charge, error)
	GetByID(ctx context.Context, input GetByIDInput) (Charge, error)
}

type RealizationRunAdapter interface {
	CreateRealizationRun(ctx context.Context, chargeID meta.ChargeID, input CreateRealizationRunInput) (RealizationRunBase, error)
	UpdateRealizationRun(ctx context.Context, input UpdateRealizationRunInput) (RealizationRunBase, error)
}

type RealizationRunCreditAllocationAdapter interface {
	CreateRunCreditRealization(ctx context.Context, runID RealizationRunID, creditAllocations creditrealization.CreateInputs) (creditrealization.Realizations, error)
}

type RealizationRunInvoiceUsageAdapter interface {
	CreateRunInvoicedUsage(ctx context.Context, runID RealizationRunID, invoicedUsage invoicedusage.AccruedUsage) (invoicedusage.AccruedUsage, error)
}

type RealizationRunPaymentAdapter interface {
	CreateRunPayment(ctx context.Context, runID RealizationRunID, in payment.InvoicedCreate) (payment.Invoiced, error)
	UpdateRunPayment(ctx context.Context, in payment.Invoiced) (payment.Invoiced, error)
}
