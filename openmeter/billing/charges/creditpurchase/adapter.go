package creditpurchase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/filter"
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
	SetResolvedCostBasis(ctx context.Context, input SetResolvedCostBasisInput) (costbasis.CostBasis, error)
	UpdateCharge(ctx context.Context, charge ChargeBase) (ChargeBase, error)
	MarkVoided(ctx context.Context, input MarkVoidedAdapterInput) (ChargeBase, error)
	GetByIDs(ctx context.Context, ids GetByIDsInput) ([]Charge, error)
	GetByID(ctx context.Context, id GetByIDInput) (Charge, error)
	ListCharges(ctx context.Context, input ListChargesInput) (pagination.Result[Charge], error)
	ListFundedCreditActivities(ctx context.Context, input ListFundedCreditActivitiesInput) (ListFundedCreditActivitiesResult, error)
}

type SetResolvedCostBasisInput struct {
	ChargeID          meta.ChargeID
	ChargeCostBasisID string
	State             costbasis.State
}

var _ models.Validator = SetResolvedCostBasisInput{}

func (i SetResolvedCostBasisInput) Validate() error {
	var errs []error

	if err := i.ChargeID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge ID: %w", err))
	}

	if i.ChargeCostBasisID == "" {
		errs = append(errs, errors.New("charge cost basis ID is required"))
	}

	if err := i.State.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("state: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type CreateChargeInput struct {
	CreateInput

	// InitialCostBasisState is persisted only for manual and pinned custom-currency
	// cost bases. Fiat state is reconstructed from immutable charge-row fields,
	// while dynamic custom-currency state is resolved at realization time.
	InitialCostBasisState *costbasis.State
}

var _ models.Validator = (*CreateChargeInput)(nil)

func (i CreateChargeInput) Validate() error {
	var errs []error

	if err := i.CreateInput.Validate(); err != nil {
		errs = append(errs, err)
	}

	switch i.Intent.Settlement.Type() {
	case SettlementTypePromotional:
		if i.InitialCostBasisState != nil {
			errs = append(errs, errors.New("promotional credit purchase cannot have initial cost-basis state"))
		}
	case SettlementTypeInvoice, SettlementTypeExternal:
		switch i.Intent.CostBasis.Type() {
		case CostBasisTypeFiat:
			if i.InitialCostBasisState != nil {
				errs = append(errs, errors.New("fiat credit purchase cannot have initial cost-basis state"))
			}
		case CostBasisTypeCustomCurrency:
			switch i.Intent.CostBasis.GetCustomCurrencyModeOrEmpty() {
			case costbasis.ModeDynamic:
				if i.InitialCostBasisState != nil {
					errs = append(errs, errors.New("dynamic credit purchase cannot have initial cost-basis state"))
				}
			case costbasis.ModeManual, costbasis.ModePinned:
				if i.InitialCostBasisState == nil {
					errs = append(errs, errors.New("manual or pinned credit purchase requires initial cost-basis state"))
				}
			}
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
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

type MarkVoidedInput struct {
	ChargeID meta.ChargeID
	VoidedAt time.Time
}

func (i MarkVoidedInput) Validate() error {
	var errs []error

	if err := i.ChargeID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge ID: %w", err))
	}
	if i.VoidedAt.IsZero() {
		errs = append(errs, errors.New("voided at is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type MarkVoidedAdapterInput struct {
	Charge   Charge
	VoidedAt time.Time
}

func (i MarkVoidedAdapterInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}
	if i.VoidedAt.IsZero() {
		errs = append(errs, errors.New("voided at is required"))
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
	Key        *filter.FilterString
	// Voided filters by whether the charge has been voided.
	Voided *bool
	// Expiration filters by whether expires_at has passed as of a point in time.
	Expiration *ListChargesExpirationFilter

	IncludeDeleted bool
	Expands        meta.Expands
}

type ListChargesExpirationFilter struct {
	AsOf    time.Time
	Expired bool
}

func (f ListChargesExpirationFilter) Validate() error {
	if f.AsOf.IsZero() {
		return errors.New("as of is required")
	}

	return nil
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

	if i.Key != nil {
		if err := i.Key.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("key: %w", err))
		}
	}

	if i.Expiration != nil {
		if err := i.Expiration.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("expiration: %w", err))
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
