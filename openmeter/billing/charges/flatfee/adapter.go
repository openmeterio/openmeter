package flatfee

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Adapter interface {
	CreateCharges(ctx context.Context, charges CreateChargesInput) ([]Charge, error)
	UpdateCharge(ctx context.Context, charge Charge) error
	GetByIDs(ctx context.Context, ids GetByIDsInput) ([]Charge, error)

	CreateInvoicedUsage(ctx context.Context, chargeID meta.ChargeID, invoicedUsage invoicedusage.AccruedUsage) (invoicedusage.AccruedUsage, error)

	CreateCreditAllocations(ctx context.Context, chargeID meta.ChargeID, creditAllocations creditrealization.CreateInputs) (creditrealization.Realizations, error)

	CreatePayment(ctx context.Context, chargeID meta.ChargeID, paymentSettlement payment.InvoicedCreate) (payment.Invoiced, error)
	UpdatePayment(ctx context.Context, paymentSettlement payment.Invoiced) (payment.Invoiced, error)

	entutils.TxCreator
}

type IntentWithInitialStatus struct {
	Intent
	InitialStatus meta.ChargeStatus
}

func (i IntentWithInitialStatus) Validate() error {
	var errs []error
	if err := i.Intent.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intent: %w", err))
	}

	if i.InitialStatus == "" {
		if err := i.InitialStatus.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("initial status: %w", err))
		}
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
