package charges

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

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

type FlatFeeState struct{}

func (s FlatFeeState) Validate() error {
	var errs []error

	return errors.Join(errs...)
}
