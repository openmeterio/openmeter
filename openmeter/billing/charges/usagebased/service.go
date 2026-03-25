package usagebased

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
)

type Service interface {
	UsageBasedService
	InvoiceLifecycleHooks
}

type UsageBasedService interface {
	Create(ctx context.Context, input CreateInput) ([]ChargeWithGatheringLine, error)
	GetByIDs(ctx context.Context, input GetByIDsInput) ([]Charge, error)
	AdvanceCharge(ctx context.Context, input AdvanceChargeInput) (*Charge, error)
}

type InvoiceLifecycleHooks interface {
	PostLineAssignedToInvoice(ctx context.Context, charge Charge, line billing.GatheringLine) (creditrealization.Realizations, error)
	PostInvoiceIssued(ctx context.Context, charge Charge, lineWithHeader billing.StandardLineWithInvoiceHeader) error
	PostInvoicePaymentAuthorized(ctx context.Context, charge Charge, lineWithHeader billing.StandardLineWithInvoiceHeader) error
	PostInvoicePaymentSettled(ctx context.Context, charge Charge, lineWithHeader billing.StandardLineWithInvoiceHeader) error
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
	IDs       []string
	Expands   meta.Expands
}

func (i GetByIDsInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if err := i.Expands.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("expands: %w", err))
	}

	return errors.Join(errs...)
}

type AdvanceChargeInput struct {
	ChargeID         meta.ChargeID
	CustomerOverride billing.CustomerOverrideWithDetails
	FeatureMeter     feature.FeatureMeter
}

func (i AdvanceChargeInput) Validate() error {
	var errs []error
	if err := i.ChargeID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge ID: %w", err))
	}

	if i.CustomerOverride.Customer == nil {
		errs = append(errs, errors.New("expanded customer is required"))
	}

	if err := i.CustomerOverride.MergedProfile.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("merged profile is required: %w", err))
	}

	if i.FeatureMeter.Meter == nil {
		errs = append(errs, errors.New("feature meter is required"))
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
