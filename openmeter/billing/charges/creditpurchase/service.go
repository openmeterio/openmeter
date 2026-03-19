package creditpurchase

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
)

type Service interface {
	CreditPurchaseService
	ExternalPaymentLifecycle
	InvoicePaymentLifecycle
}

type CreditPurchaseService interface {
	// Create creates a new credit purchase charge. It can only handle a single intent at a time
	// as based on current state of credits we are not going to create multiple credit purchases at once.
	Create(ctx context.Context, input CreateInput) (ChargeWithGatheringLine, error)

	GetByMetas(ctx context.Context, input GetByMetasInput) ([]Charge, error)

	// PostLineAssignedToInvoice creates the initial InvoiceSettlement (payment.Invoiced) record
	// when a credit purchase gathering line is assigned to a standard invoice.
	PostLineAssignedToInvoice(ctx context.Context, input PostLineAssignedToInvoiceInput) error
}

type PostLineAssignedToInvoiceInput struct {
	Charge    Charge
	LineID    string
	InvoiceID string
}

func (i PostLineAssignedToInvoiceInput) Validate() error {
	var errs []error

	if i.LineID == "" {
		errs = append(errs, fmt.Errorf("line ID is required"))
	}

	if i.InvoiceID == "" {
		errs = append(errs, fmt.Errorf("invoice ID is required"))
	}

	return errors.Join(errs...)
}

type ChargeWithGatheringLine struct {
	Charge                Charge
	GatheringLineToCreate *billing.GatheringLine
}

type ExternalPaymentLifecycle interface {
	HandleExternalPaymentAuthorized(ctx context.Context, charge Charge) (Charge, error)
	HandleExternalPaymentSettled(ctx context.Context, charge Charge) (Charge, error)
}

type InvoicePaymentLifecycle interface {
	HandleInvoicePaymentAuthorized(ctx context.Context, charge Charge, lineWithHeader billing.StandardLineWithInvoiceHeader) error
	HandleInvoicePaymentSettled(ctx context.Context, charge Charge, lineWithHeader billing.StandardLineWithInvoiceHeader) error
}

type CreateInput struct {
	Namespace string
	Intent    Intent
}

func (i CreateInput) Validate() error {
	var errs []error
	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if err := i.Intent.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intent: %w", err))
	}

	return errors.Join(errs...)
}

type GetByMetasInput struct {
	Namespace string
	Expands   meta.Expands
	Charges   meta.Charges
}

func (i GetByMetasInput) Validate() error {
	var errs []error
	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if err := i.Charges.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charges: %w", err))
	}

	if err := i.Expands.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("expands: %w", err))
	}

	return errors.Join(errs...)
}
