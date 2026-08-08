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
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// CustomerChargeAPIService is the API-facing facade for customer-scoped charge operations.
type CustomerChargeAPIService interface {
	CreateCustomerCharge(ctx context.Context, input CreateCustomerChargeInput) (Charge, error)
	DeleteCustomerCharge(ctx context.Context, input DeleteCustomerChargeInput) error
	SetCustomerChargeOverride(ctx context.Context, input SetCustomerChargeOverrideInput) (Charge, error)
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
