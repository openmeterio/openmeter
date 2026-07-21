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

// CustomerChargeAPIService resolves API-facing references before constructing
// complete charge intents for the core charge creation flow.
type CustomerChargeAPIService interface {
	CreateCustomerCharge(ctx context.Context, input CreateCustomerChargeInput) (Charge, error)
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
	FeatureKey          *string
	SettlementMode      productcatalog.SettlementMode
}

type CreateCustomerChargeUsageBasedInput struct {
	IntentMutableFields usagebased.IntentMutableFields
	FeatureKey          string
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
