package flatfee

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ meta.ChargeAccessor = (*Charge)(nil)

type Charge struct {
	meta.ManagedResource

	Intent Intent            `json:"intent"`
	Status meta.ChargeStatus `json:"status"`

	State State `json:"state"`
}

func (c Charge) Validate() error {
	var errs []error

	if err := c.ManagedResource.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("managed resource: %w", err))
	}

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

func (c Charge) GetChargeID() meta.ChargeID {
	return meta.ChargeID{
		Namespace: c.Namespace,
		ID:        c.ID,
	}
}

type Intent struct {
	meta.Intent

	InvoiceAt      time.Time                     `json:"invoiceAt"`
	SettlementMode productcatalog.SettlementMode `json:"settlementMode"`

	PaymentTerm         productcatalog.PaymentTermType     `json:"paymentTerm"`
	FeatureKey          string                             `json:"featureKey,omitempty"`
	PercentageDiscounts *productcatalog.PercentageDiscount `json:"percentageDiscounts"`

	ProRating             productcatalog.ProRatingConfig `json:"proRating"`
	AmountBeforeProration alpacadecimal.Decimal          `json:"amountBeforeProration"`
	AmountAfterProration  alpacadecimal.Decimal          `json:"amountAfterProration"`
}

func (i Intent) Validate() error {
	var errs []error

	if err := i.Intent.Validate(); err != nil {
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

func (c Charge) ErrorAttributes() models.Attributes {
	return models.Attributes{
		"charge_id":   c.ID,
		"namespace":   c.Namespace,
		"charge_type": string(meta.ChargeTypeFlatFee),
	}
}

type State struct {
	CreditRealizations creditrealization.Realizations `json:"creditRealizations"`
	AccruedUsage       *invoicedusage.AccruedUsage    `json:"accruedUsage"`
	Payment            *payment.Invoiced              `json:"payment"`
}

func (s State) Validate() error {
	var errs []error

	for _, realization := range s.CreditRealizations {
		if err := realization.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("credit realization[id=%s]: %w", realization.ID, err))
		}
	}

	if s.Payment != nil {
		if err := s.Payment.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("payment[id=%s]: %w", s.Payment.ID, err))
		}
	}

	if s.AccruedUsage != nil {
		if err := s.AccruedUsage.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("accrued usage[id=%s]: %w", s.AccruedUsage.ID, err))
		}
	}

	return errors.Join(errs...)
}
