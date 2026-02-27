package charges

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

var _ ChargeAccessor = (*FlatFeeCharge)(nil)

type FlatFeeCharge struct {
	ManagedResource

	Intent FlatFeeIntent `json:"intent"`
	Status ChargeStatus  `json:"status"`

	State FlatFeeState `json:"state"`
}

func (c FlatFeeCharge) AsCharge() Charge {
	return Charge{
		t:       ChargeTypeFlatFee,
		flatFee: &c,
	}
}

func (c FlatFeeCharge) Validate() error {
	var errs []error

	if err := c.Intent.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intent: %w", err))
	}

	if err := c.Status.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("status: %w", err))
	}

	if err := c.State.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("state: %w", err))
	}

	return errors.Join(errs...)
}

func (c FlatFeeCharge) GetChargeID() ChargeID {
	return ChargeID{
		Namespace: c.Namespace,
		ID:        c.ID,
	}
}

type FlatFeeIntent struct {
	IntentMeta

	InvoiceAt      time.Time                     `json:"invoiceAt"`
	SettlementMode productcatalog.SettlementMode `json:"settlementMode"`

	PaymentTerm         productcatalog.PaymentTermType     `json:"paymentTerm"`
	FeatureKey          string                             `json:"featureKey,omitempty"`
	PercentageDiscounts *productcatalog.PercentageDiscount `json:"percentageDiscounts"`

	ProRating             productcatalog.ProRatingConfig `json:"proRating"`
	AmountBeforeProration alpacadecimal.Decimal          `json:"amountBeforeProration"`
	AmountAfterProration  alpacadecimal.Decimal          `json:"amountAfterProration"`
}

func (i FlatFeeIntent) Validate() error {
	var errs []error

	if err := i.IntentMeta.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := i.SettlementMode.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("settlement mode: %w", err))
	}

	if i.AmountBeforeProration.IsNegative() {
		errs = append(errs, fmt.Errorf("amount before proration cannot be negative"))
	}

	if !slices.Contains(productcatalog.PaymentTermType("").Values(), string(i.PaymentTerm)) {
		errs = append(errs, fmt.Errorf("invalid payment term %s", i.PaymentTerm))
	}

	if i.InvoiceAt.IsZero() {
		errs = append(errs, fmt.Errorf("invoice at is required"))
	}

	if i.PercentageDiscounts != nil {
		if err := i.PercentageDiscounts.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("percentage discounts: %w", err))
		}
	}

	if err := i.ProRating.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("pro rating: %w", err))
	}

	if i.AmountAfterProration.IsNegative() {
		errs = append(errs, fmt.Errorf("amount after proration cannot be negative"))
	}

	return errors.Join(errs...)
}

type FlatFeeState struct {
	CreditRealizations    CreditRealizations               `json:"creditRealizations"`
	SettledTransaction    *LedgerTransactionGroupReference `json:"settledTransaction"`
	AuthorizedTransaction *LedgerTransactionGroupReference `json:"authorizedTransaction"`
}

func (s FlatFeeState) Validate() error {
	var errs []error

	for _, realization := range s.CreditRealizations {
		if err := realization.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("credit realization: %w", err))
		}
	}

	return errors.Join(errs...)
}

type FlatFeeService interface {
	PostCreate(ctx context.Context, charge FlatFeeCharge) (PostCreateFlatFeeResult, error)
	PostLineAssignedToInvoice(ctx context.Context, charge FlatFeeCharge, line billing.GatheringLine) (CreditRealizations, error)
	PostPaymentAuthorized(ctx context.Context, charge FlatFeeCharge, lineWithHeader billing.StandardLineWithInvoiceHeader) error
	PostPaymentSettled(ctx context.Context, charge FlatFeeCharge, lineWithHeader billing.StandardLineWithInvoiceHeader) error
}

type PostCreateFlatFeeResult struct {
	Charge                FlatFeeCharge
	GatheringLineToCreate *billing.GatheringLine
}

type OnFlatFeeAssignedToInvoiceInput struct {
	Charge            FlatFeeCharge         `json:"charge"`
	ServicePeriod     timeutil.ClosedPeriod `json:"servicePeriod"`
	PreTaxTotalAmount alpacadecimal.Decimal `json:"totalAmount"`
}

func (i OnFlatFeeAssignedToInvoiceInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if err := i.ServicePeriod.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("service period: %w", err))
	}

	if i.PreTaxTotalAmount.IsNegative() {
		errs = append(errs, fmt.Errorf("pre tax total amount cannot be negative"))
	}

	return errors.Join(errs...)
}

type FlatFeeHandler interface {
	// OnFlatFeeAssignedToInvoice is called when a flat fee is being assigned to an invoice
	OnFlatFeeAssignedToInvoice(ctx context.Context, input OnFlatFeeAssignedToInvoiceInput) ([]CreditRealizationCreateInput, error)

	// OnFlatFeePaymentAuthorized is called when a flat fee payment is authorized
	OnFlatFeePaymentAuthorized(ctx context.Context, charge FlatFeeCharge) (LedgerTransactionGroupReference, error)

	// OnFlatFeePaymentSettled is called when a flat fee payment is settled
	OnFlatFeePaymentSettled(ctx context.Context, charge FlatFeeCharge) (LedgerTransactionGroupReference, error)

	// OnFlatFeePaymentUncollectible is called when a flat fee payment is uncollectible
	OnFlatFeePaymentUncollectible(ctx context.Context, charge FlatFeeCharge) (LedgerTransactionGroupReference, error)
}
