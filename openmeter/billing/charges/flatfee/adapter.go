package flatfee

import (
	"context"
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Adapter interface {
	CreateCharges(ctx context.Context, charges CreateChargesInput) ([]Charge, error)
	UpdateCharge(ctx context.Context, charge Charge) error
	DeleteCharge(ctx context.Context, charge Charge) error
	GetByIDs(ctx context.Context, ids GetByIDsInput) ([]Charge, error)
	GetByID(ctx context.Context, id GetByIDInput) (Charge, error)

	CreateInvoicedUsage(ctx context.Context, chargeID meta.ChargeID, invoicedUsage invoicedusage.AccruedUsage) (invoicedusage.AccruedUsage, error)

	CreateCreditAllocations(ctx context.Context, chargeID meta.ChargeID, creditAllocations creditrealization.CreateInputs) (creditrealization.Realizations, error)

	CreatePayment(ctx context.Context, chargeID meta.ChargeID, paymentSettlement payment.InvoicedCreate) (payment.Invoiced, error)
	UpdatePayment(ctx context.Context, paymentSettlement payment.Invoiced) (payment.Invoiced, error)

	entutils.TxCreator
}

type IntentWithInitialStatus struct {
	Intent
	InitialStatus        meta.ChargeStatus
	AmountAfterProration alpacadecimal.Decimal
}

func (i IntentWithInitialStatus) Validate() error {
	var errs []error
	if err := i.Intent.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intent: %w", err))
	}

	if i.AmountAfterProration.IsNegative() {
		errs = append(errs, fmt.Errorf("amount after proration cannot be negative"))
	}

	// Initial status is optional, but if it is set, it must be valid
	if i.InitialStatus != "" {
		if err := i.InitialStatus.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("initial status: %w", err))
		}
	}
	return errors.Join(errs...)
}

type GetByIDsInput struct {
	Namespace string
	IDs       []string

	Expands meta.Expands
}

func (i GetByIDsInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	for _, id := range i.IDs {
		if id == "" {
			errs = append(errs, errors.New("id is required"))
		}
	}

	if err := i.Expands.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("expands: %w", err))
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

type CreateChargesInput struct {
	Namespace string
	Intents   []IntentWithInitialStatus
}

func (i CreateChargesInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	for idx, intent := range i.Intents {
		if err := intent.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("intent [%d]: %w", idx, err))
		}
	}

	return errors.Join(errs...)
}
