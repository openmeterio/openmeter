package usagebased

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
)

type Service interface {
	UsageBasedService
	// InvoiceLifecycleHooks
}

type UsageBasedService interface {
	Create(ctx context.Context, input CreateInput) ([]ChargeWithGatheringLine, error)
	GetByIDs(ctx context.Context, input GetByIDsInput) ([]Charge, error)
}

type InvoiceLifecycleHooks interface {
	PostLineAssignedToInvoice(ctx context.Context, charge Charge, line billing.GatheringLine) (creditrealization.Realizations, error)
	PostInvoiceIssued(ctx context.Context, charge Charge, lineWithHeader billing.StandardLineWithInvoiceHeader) error
	PostPaymentAuthorized(ctx context.Context, charge Charge, lineWithHeader billing.StandardLineWithInvoiceHeader) error
	PostPaymentSettled(ctx context.Context, charge Charge, lineWithHeader billing.StandardLineWithInvoiceHeader) error
}

type CreateInput struct {
	Namespace string
	Intents   []Intent
}

func (i CreateInput) Validate() error {
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

type ChargeWithGatheringLine struct {
	Charge                Charge
	GatheringLineToCreate *billing.GatheringLine
}

type GetByIDsInput struct {
	Namespace string
	Expands   meta.Expands
	Charges   meta.Charges
}

func (i GetByIDsInput) Validate() error {
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
