package meta

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

type Adapter interface {
	UpdateStatus(ctx context.Context, in UpdateStatusInput) (Charge, error)
	Create(ctx context.Context, in CreateInput) (Charges, error)

	GetByIDs(ctx context.Context, ids GetByIDsInput) (Charges, error)
	ListByCustomer(ctx context.Context, input ListByCustomerInput) (Charges, error)

	entutils.TxCreator
}

type UpdateStatusInput struct {
	ChargeID     ChargeID
	Status       ChargeStatus
	AdvanceAfter *time.Time
}

func (i UpdateStatusInput) Validate() error {
	var errs []error

	if err := i.ChargeID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge ID: %w", err))
	}

	if err := i.Status.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("status: %w", err))
	}

	return errors.Join(errs...)
}

type CreateInput struct {
	Namespace string
	Intents   []IntentCreate
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

type IntentCreate struct {
	Intent

	Type          ChargeType   `json:"type"`
	InitialStatus ChargeStatus `json:"status"`
	AdvanceAfter  *time.Time   `json:"advanceAfter"`
}

func (i IntentCreate) Validate() error {
	var errs []error

	if err := i.Intent.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intent: %w", err))
	}

	if err := i.Type.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("type: %w", err))
	}

	if i.InitialStatus != "" {
		if err := i.InitialStatus.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("initial status: %w", err))
		}
	}

	return errors.Join(errs...)
}

type GetByIDsInput struct {
	Namespace string
	ChargeIDs []string
}

func (i GetByIDsInput) Validate() error {
	var errs []error
	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	for idx, id := range i.ChargeIDs {
		if id == "" {
			errs = append(errs, fmt.Errorf("charge ID [%d]: cannot be empty", idx))
		}
	}

	return errors.Join(errs...)
}

type ListByCustomerInput struct {
	Customer customer.CustomerID
}

func (i ListByCustomerInput) Validate() error {
	var errs []error

	if err := i.Customer.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("customer: %w", err))
	}

	return errors.Join(errs...)
}
