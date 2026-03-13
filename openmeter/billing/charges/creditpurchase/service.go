package creditpurchase

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
)

type Service interface {
	CreditPurchaseService
	ExternalPaymentLifecycle
}

type CreditPurchaseService interface {
	// Create creates a new credit purchase charge. It can only handle a single intent at a time
	// as based on current state of credits we are not going to create multiple credit purchases at once.
	Create(ctx context.Context, input CreateInput) (Charge, error)

	GetByIDs(ctx context.Context, input GetByIDsInput) ([]Charge, error)
}

type ExternalPaymentLifecycle interface {
	HandleExternalPaymentAuthorized(ctx context.Context, charge Charge) (Charge, error)
	HandleExternalPaymentSettled(ctx context.Context, charge Charge) (Charge, error)
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
