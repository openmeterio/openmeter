package creditpurchase

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type Adapter interface {
	entutils.TxCreator

	UpdateCharge(ctx context.Context, charge Charge) (Charge, error)
	CreateCharge(ctx context.Context, in CreateChargeInput) (Charge, error)
	GetByIDs(ctx context.Context, ids GetByIDsInput) ([]Charge, error)

	CreateExternalPayment(ctx context.Context, chargeID meta.ChargeID, payment payment.ExternalCreateInput) (payment.External, error)
	UpdateExternalPayment(ctx context.Context, payment payment.External) (payment.External, error)

	CreateInvoicedPayment(ctx context.Context, chargeID meta.ChargeID, payment payment.InvoicedCreate) (payment.Invoiced, error)
	UpdateInvoicedPayment(ctx context.Context, payment payment.Invoiced) (payment.Invoiced, error)
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

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type CreateChargeInput struct {
	Namespace string
	Intent    Intent
}

func (i CreateChargeInput) Validate() error {
	var errs []error
	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if err := i.Intent.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intent: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
