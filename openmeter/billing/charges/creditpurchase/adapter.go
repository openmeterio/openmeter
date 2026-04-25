package creditpurchase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Adapter interface {
	ChargeAdapter
	CreditGrantAdapter
	ExternalPaymentAdapter
	InvoicedPaymentAdapter

	entutils.TxCreator
}

type ChargeAdapter interface {
	CreateCharge(ctx context.Context, in CreateChargeInput) (Charge, error)
	UpdateCharge(ctx context.Context, charge ChargeBase) (ChargeBase, error)
	GetByIDs(ctx context.Context, ids GetByIDsInput) ([]Charge, error)
	GetByID(ctx context.Context, id GetByIDInput) (Charge, error)
	ListCharges(ctx context.Context, input ListChargesInput) (pagination.Result[Charge], error)
	ListFundedCreditActivities(ctx context.Context, input ListFundedCreditActivitiesInput) (ListFundedCreditActivitiesResult, error)
}

type ExternalPaymentAdapter interface {
	CreateExternalPayment(ctx context.Context, chargeID meta.ChargeID, payment payment.ExternalCreateInput) (payment.External, error)
	UpdateExternalPayment(ctx context.Context, payment payment.External) (payment.External, error)
}

type CreditGrantAdapter interface {
	CreateCreditGrant(ctx context.Context, chargeID meta.ChargeID, input CreateCreditGrantInput) (ledgertransaction.TimedGroupReference, error)
}

type InvoicedPaymentAdapter interface {
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

type CreateCreditGrantInput struct {
	TransactionGroupID string
	GrantedAt          time.Time
}

func (i CreateCreditGrantInput) Validate() error {
	var errs []error

	if i.TransactionGroupID == "" {
		errs = append(errs, errors.New("transaction group ID is required"))
	}

	if i.GrantedAt.IsZero() {
		errs = append(errs, errors.New("granted at is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
