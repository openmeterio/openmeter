package charges

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// CustomerChargeAPIService is the API-facing facade for customer-scoped charge operations.
type CustomerChargeAPIService interface {
	CreateCustomerCharge(ctx context.Context, input CreateCustomerChargeInput) (CustomerCharge, error)
	DeleteCustomerCharge(ctx context.Context, input DeleteCustomerChargeInput) error
	SetCustomerChargeOverride(ctx context.Context, input SetCustomerChargeOverrideInput) (Charge, error)
	ClearCustomerChargeOverride(ctx context.Context, input ClearCustomerChargeOverrideInput) (Charge, error)
	// ListCustomerCharges lists charges with API-facing expand resolution: it
	// attaches the resolved realization view to each charge and side-loads the
	// referenced entities the applied expands ask for.
	ListCustomerCharges(ctx context.Context, input ListCustomerChargesInput) (ListCustomerChargesResult, error)
}

// SubscriptionService is the narrow subscription dependency of the
// customer-charge API facade; subscription.Service satisfies it.
type SubscriptionService interface {
	List(ctx context.Context, input subscription.ListSubscriptionsInput) (subscription.SubscriptionList, error)
}

type ListCustomerChargesInput struct {
	ListChargesInput
}

func (i ListCustomerChargesInput) Validate() error {
	var errs []error

	if err := i.ListChargesInput.Validate(); err != nil {
		errs = append(errs, err)
	}

	if len(i.CustomerIDs) != 1 {
		errs = append(errs, errors.New("exactly one customer ID is required"))
	}

	// The customer-charge API only serves flat fee and usage based charges;
	// credit purchases belong to the credit grants API. Rejecting other types
	// here avoids failing a whole page at conversion time.
	if len(i.ChargeTypes) == 0 {
		errs = append(errs, errors.New("at least one charge type is required"))
	}

	for _, chargeType := range i.ChargeTypes {
		if chargeType != meta.ChargeTypeFlatFee && chargeType != meta.ChargeTypeUsageBased {
			errs = append(errs, fmt.Errorf("unsupported charge type for customer charge listing: %s", chargeType))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// ListCustomerChargesResult is the paginated charges together with the
// expands the facade applied; expanded entities are carried by each
// CustomerCharge.
type ListCustomerChargesResult struct {
	Charges pagination.Result[CustomerCharge]
	Expands meta.Expands
}

type CreditPurchaseFacadeService interface {
	HandleCreditPurchaseExternalPaymentStateTransition(ctx context.Context, input HandleCreditPurchaseExternalPaymentStateTransitionInput) (creditpurchase.Charge, error)
}

type CreateCustomerChargeInput struct {
	Namespace         string
	CustomerID        string
	CurrencyCode      currencyx.Code
	TaxConfig         productcatalog.TaxCodeConfig
	UniqueReferenceID *string

	FlatFee    *CreateCustomerChargeFlatFeeInput
	UsageBased *CreateCustomerChargeUsageBasedInput
}

type CreateCustomerChargeFlatFeeInput struct {
	IntentMutableFields flatfee.IntentMutableFields
	FeatureID           *string
	SettlementMode      productcatalog.SettlementMode
}

type CreateCustomerChargeUsageBasedInput struct {
	IntentMutableFields usagebased.IntentMutableFields
	FeatureID           string
	SettlementMode      productcatalog.SettlementMode
}

func (i CreateCustomerChargeInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.CustomerID == "" {
		errs = append(errs, errors.New("customer ID is required"))
	}

	if err := i.CurrencyCode.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency code: %w", err))
	}

	if (i.FlatFee == nil) == (i.UsageBased == nil) {
		errs = append(errs, errors.New("exactly one charge type is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type DeleteCustomerChargeInput struct {
	Namespace  string
	CustomerID string
	ChargeID   string

	// PaymentAdjustment is required so callers explicitly select how deletion
	// compensates already-realized economic effects.
	PaymentAdjustment PaymentAdjustment
}

type PaymentAdjustment string

const (
	// PaymentAdjustmentNone requests no compensating payment adjustment. Charge
	// deletion still performs its normal invoice lifecycle reconciliation.
	PaymentAdjustmentNone PaymentAdjustment = "none"
)

func (a PaymentAdjustment) Validate() error {
	if a != PaymentAdjustmentNone {
		return models.NewGenericValidationError(fmt.Errorf("invalid payment adjustment: %s", a))
	}

	return nil
}

func (i DeleteCustomerChargeInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.CustomerID == "" {
		errs = append(errs, errors.New("customer ID is required"))
	}

	if i.ChargeID == "" {
		errs = append(errs, errors.New("charge ID is required"))
	}

	if err := i.PaymentAdjustment.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("payment adjustment: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// SetCustomerChargeOverrideInput replaces the complete mutable override layer
// of a flat-fee or usage-based charge. The immutable base intent is preserved.
type SetCustomerChargeOverrideInput struct {
	Namespace  string
	CustomerID string
	ChargeID   string

	FlatFee    *flatfee.IntentMutableFields
	UsageBased *usagebased.IntentMutableFields
}

func (i SetCustomerChargeOverrideInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.CustomerID == "" {
		errs = append(errs, errors.New("customer ID is required"))
	}

	if i.ChargeID == "" {
		errs = append(errs, errors.New("charge ID is required"))
	}

	if (i.FlatFee == nil) == (i.UsageBased == nil) {
		errs = append(errs, errors.New("exactly one charge type is required"))
	}

	if i.FlatFee != nil {
		if i.FlatFee.IntentDeletedAt != nil {
			errs = append(errs, errors.New("flat fee intent deleted at cannot be set by an override update"))
		}

		if err := i.FlatFee.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("flat fee intent mutable fields: %w", err))
		}
	}

	if i.UsageBased != nil {
		if i.UsageBased.IntentDeletedAt != nil {
			errs = append(errs, errors.New("usage based intent deleted at cannot be set by an override update"))
		}

		if err := i.UsageBased.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("usage based intent mutable fields: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// ClearCustomerChargeOverrideInput removes the manual override layer from a
// flat-fee or usage-based charge and makes its base intent effective again.
type ClearCustomerChargeOverrideInput struct {
	Namespace  string
	CustomerID string
	ChargeID   string
}

func (i ClearCustomerChargeOverrideInput) Validate() error {
	var errs []error

	if i.Namespace == "" {
		errs = append(errs, errors.New("namespace is required"))
	}

	if i.CustomerID == "" {
		errs = append(errs, errors.New("customer ID is required"))
	}

	if i.ChargeID == "" {
		errs = append(errs, errors.New("charge ID is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type HandleCreditPurchaseExternalPaymentStateTransitionInput struct {
	ChargeID meta.ChargeID

	TargetPaymentState payment.Status
}

func (i HandleCreditPurchaseExternalPaymentStateTransitionInput) Validate() error {
	var errs []error

	if err := i.ChargeID.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge ID: %w", err))
	}

	if err := i.TargetPaymentState.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("target payment state: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
