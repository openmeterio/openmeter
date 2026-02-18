package charges

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type IntentMeta struct {
	Metadata    models.Metadata              `json:"metadata"`
	Annotations models.Annotations           `json:"annotations"`
	ManagedBy   billing.InvoiceLineManagedBy `json:"managedBy"`
	CustomerID  string                       `json:"customerID"`

	Currency currencyx.Code `json:"currency"`

	ServicePeriod     timeutil.ClosedPeriod `json:"servicePeriod"`
	FullServicePeriod timeutil.ClosedPeriod `json:"fullServicePeriod"`
	BillingPeriod     timeutil.ClosedPeriod `json:"billingPeriod"`

	InvoiceAt time.Time `json:"invoiceAt"`

	TaxConfig         *productcatalog.TaxConfig `json:"taxConfig"`
	UniqueReferenceID *string                   `json:"childUniqueReferenceID"`

	Subscription *SubscriptionReference `json:"subscription"`
}

func (i IntentMeta) Validate() error {
	var errs []error

	if !slices.Contains(billing.InvoiceLineManagedBy("").Values(), string(i.ManagedBy)) {
		errs = append(errs, fmt.Errorf("invalid managed by %s", i.ManagedBy))
	}

	if i.CustomerID == "" {
		errs = append(errs, fmt.Errorf("customer ID is required"))
	}

	if err := i.Currency.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency: %w", err))
	}

	if err := i.ServicePeriod.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("service period: %w", err))
	}

	if err := i.FullServicePeriod.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("full service period: %w", err))
	}

	if err := i.BillingPeriod.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("billing period: %w", err))
	}

	if i.TaxConfig != nil {
		if err := i.TaxConfig.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("tax config: %w", err))
		}
	}

	if i.Subscription != nil {
		if err := i.Subscription.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("subscription: %w", err))
		}
	}

	if i.UniqueReferenceID != nil && *i.UniqueReferenceID == "" {
		errs = append(errs, fmt.Errorf("unique reference ID cannot be empty"))
	}

	return errors.Join(errs...)
}

type FlatFeeIntent struct {
	PaymentTerm         productcatalog.PaymentTermType     `json:"paymentTerm"`
	FeatureKey          string                             `json:"featureKey,omitempty"`
	PercentageDiscounts *productcatalog.PercentageDiscount `json:"percentageDiscounts"`

	ProRating             productcatalog.ProRatingConfig `json:"proRating"`
	AmountBeforeProration alpacadecimal.Decimal          `json:"amountBeforeProration"`
	AmountAfterProration  alpacadecimal.Decimal          `json:"amountAfterProration"`
}

func (i FlatFeeIntent) ValidateWithMeta(meta IntentMeta) error {
	var errs []error

	if i.AmountBeforeProration.IsNegative() {
		errs = append(errs, fmt.Errorf("amount before proration cannot be negative"))
	}

	if !slices.Contains(productcatalog.PaymentTermType("").Values(), string(i.PaymentTerm)) {
		errs = append(errs, fmt.Errorf("invalid payment term %s", i.PaymentTerm))
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

type UsageBasedIntent struct {
	Price      productcatalog.Price `json:"price"`
	FeatureKey string               `json:"featureKey,omitempty"`

	Discounts *productcatalog.Discounts `json:"rateCardDiscounts"`
}

func (i UsageBasedIntent) ValidateWithMeta(meta IntentMeta) error {
	var errs []error

	if meta.InvoiceAt.IsZero() || meta.InvoiceAt.Before(meta.ServicePeriod.From) {
		errs = append(errs, fmt.Errorf("invoice at must be after service period from"))
	}

	if err := i.Price.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("price: %w", err))
	}

	if err := i.Discounts.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("discounts: %w", err))
	}

	if i.FeatureKey == "" {
		errs = append(errs, fmt.Errorf("feature key is required"))
	}

	return errors.Join(errs...)
}

type SubscriptionReference struct {
	SubscriptionID string `json:"subscriptionID"`
	PhaseID        string `json:"phaseID"`
	ItemID         string `json:"itemID"`
}

func (r SubscriptionReference) Validate() error {
	var errs []error

	if r.SubscriptionID == "" {
		errs = append(errs, fmt.Errorf("subscription ID is required"))
	}

	if r.PhaseID == "" {
		errs = append(errs, fmt.Errorf("phase ID is required"))
	}

	if r.ItemID == "" {
		errs = append(errs, fmt.Errorf("item ID is required"))
	}

	return errors.Join(errs...)
}

type IntentType string

const (
	IntentTypeFlatFee    IntentType = "flat_fee"
	IntentTypeUsageBased IntentType = "usage_based"
)

func (t IntentType) Values() []string {
	return []string{
		string(IntentTypeFlatFee),
		string(IntentTypeUsageBased),
	}
}

type Intent struct {
	IntentMeta
	IntentType IntentType

	flatFee    *FlatFeeIntent
	usageBased *UsageBasedIntent
}

func (i Intent) Validate() error {
	var errs []error

	if err := i.IntentMeta.Validate(); err != nil {
		errs = append(errs, err)
	}

	switch i.IntentType {
	case IntentTypeFlatFee:
		if i.flatFee == nil {
			errs = append(errs, fmt.Errorf("flat fee intent is required"))
		}

		if err := i.flatFee.ValidateWithMeta(i.IntentMeta); err != nil {
			errs = append(errs, err)
		}
	case IntentTypeUsageBased:
		if i.usageBased == nil {
			errs = append(errs, fmt.Errorf("usage based intent is required"))
		}

		if err := i.usageBased.ValidateWithMeta(i.IntentMeta); err != nil {
			errs = append(errs, err)
		}
	default:
		errs = append(errs, fmt.Errorf("invalid intent type: %s", i.IntentType))
	}

	return errors.Join(errs...)
}

func (i Intent) GetFlatFeeIntent() (*FlatFeeIntent, error) {
	if i.IntentType != IntentTypeFlatFee {
		return nil, fmt.Errorf("intent is not a flat fee intent")
	}

	return i.flatFee, nil
}

func (i Intent) GetUsageBasedIntent() (*UsageBasedIntent, error) {
	if i.IntentType != IntentTypeUsageBased {
		return nil, fmt.Errorf("intent is not a usage based intent")
	}

	return i.usageBased, nil
}

func NewIntent[T FlatFeeIntent | UsageBasedIntent](meta IntentMeta, v T) Intent {
	switch intent := any(v).(type) {
	case FlatFeeIntent:
		return Intent{
			IntentMeta: meta,
			IntentType: IntentTypeFlatFee,
			flatFee:    &intent,
		}
	case UsageBasedIntent:
		return Intent{
			IntentMeta: meta,
			IntentType: IntentTypeUsageBased,
			usageBased: &intent,
		}
	}

	return Intent{}
}
