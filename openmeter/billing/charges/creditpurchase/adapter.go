package creditpurchase

import (
	"context"
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Adapter interface {
	entutils.TxCreator

	UpdateCharge(ctx context.Context, charge Charge) (Charge, error)
	CreateCharge(ctx context.Context, in CreateChargeInput) (Charge, error)
	GetByIDs(ctx context.Context, ids GetByIDsInput) ([]Charge, error)
	ListCharges(ctx context.Context, input ListChargesInput) (pagination.Result[Charge], error)

	CreateExternalPayment(ctx context.Context, chargeID meta.ChargeID, payment payment.ExternalCreateInput) (payment.External, error)
	UpdateExternalPayment(ctx context.Context, payment payment.External) (payment.External, error)

	CreateInvoicedPayment(ctx context.Context, chargeID meta.ChargeID, payment payment.InvoicedCreate) (payment.Invoiced, error)
	UpdateInvoicedPayment(ctx context.Context, payment payment.Invoiced) (payment.Invoiced, error)

	BackfillAdvanceLineageSegments(ctx context.Context, input BackfillAdvanceLineageSegmentsInput) error
}

type BackfillAdvanceLineageSegmentsInput struct {
	Namespace                 string
	CustomerID                string
	Currency                  currencyx.Code
	Amount                    alpacadecimal.Decimal
	BackingTransactionGroupID string
}

func (i BackfillAdvanceLineageSegmentsInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.CustomerID == "" {
		errs = append(errs, errors.New("customer id is required"))
	}

	if err := i.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}

	if !i.Amount.IsPositive() {
		errs = append(errs, errors.New("amount must be positive"))
	}

	if i.BackingTransactionGroupID == "" {
		errs = append(errs, errors.New("backing transaction group id is required"))
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

type ListChargesInput struct {
	pagination.Page

	Namespace   string
	CustomerIDs []string

	// Optional filters
	Statuses   []meta.ChargeStatus
	Currencies []currencyx.Code

	IncludeDeleted bool
	Expands        meta.Expands
}

func (i ListChargesInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	for _, customerID := range i.CustomerIDs {
		if customerID == "" {
			errs = append(errs, errors.New("customer id is required"))
		}
	}

	for _, status := range i.Statuses {
		if err := status.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("status: %w", err))
		}
	}

	for _, currency := range i.Currencies {
		if err := currency.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("currency: %w", err))
		}
	}

	if err := i.Expands.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("expands: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
