package charges

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/customer"
)

type Service interface {
	ChargeService
	InvoiceService

	// Facade interfaces provide convinience helpers for the API layer.
	CreditPurchaseFacadeService
}

type ChargeService interface {
	GetByID(ctx context.Context, input GetByIDInput) (Charge, error)
	GetByIDs(ctx context.Context, input GetByIDsInput) (Charges, error)
	Create(ctx context.Context, input CreateInput) (Charges, error)

	AdvanceCharges(ctx context.Context, input AdvanceChargesInput) (Charges, error)
}

// InvoiceService contains methods that are over time deprecate the current billing methods.
type InvoiceService interface {
	InvoicePendingLines(ctx context.Context, input billing.InvoicePendingLinesInput) ([]billing.StandardInvoice, error)
}

type CreditPurchaseFacadeService interface {
	HandleCreditPurchaseExternalPaymentStateTransition(ctx context.Context, input HandleCreditPurchaseExternalPaymentStateTransitionInput) (creditpurchase.Charge, error)
}

type CreateInput struct {
	Namespace string
	Intents   ChargeIntents
}

func (i CreateInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, fmt.Errorf("namespace is required"))
	}

	if err := i.Intents.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intents: %w", err))
	}

	return errors.Join(errs...)
}

type GetByIDInput struct {
	ChargeID meta.ChargeID
	Expands  meta.Expands
}

func (i GetByIDInput) Validate() error {
	var errs []error

	if err := i.ChargeID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge ID: %w", err))
	}

	if err := i.Expands.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("expands: %w", err))
	}

	return errors.Join(errs...)
}

type GetByIDsInput struct {
	Namespace string
	ChargeIDs []string
	Expands   meta.Expands
}

func (i GetByIDsInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, fmt.Errorf("namespace is required"))
	}

	for idx, id := range i.ChargeIDs {
		if id == "" {
			errs = append(errs, fmt.Errorf("charge ID [%d]: cannot be empty", idx))
		}
	}

	if err := i.Expands.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("expands: %w", err))
	}

	return errors.Join(errs...)
}

type HandleCreditPurchaseExternalPaymentStateTransitionInput struct {
	ChargeID meta.ChargeID

	TargetPaymentState payment.Status
}

func (i HandleCreditPurchaseExternalPaymentStateTransitionInput) Validate() error {
	var errs []error

	if err := i.ChargeID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge ID: %w", err))
	}

	if err := i.TargetPaymentState.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("target payment state: %w", err))
	}

	return errors.Join(errs...)
}

type AdvanceChargesInput struct {
	Customer customer.CustomerID
}

func (i AdvanceChargesInput) Validate() error {
	var errs []error
	if err := i.Customer.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer ID: %w", err))
	}

	return errors.Join(errs...)
}
