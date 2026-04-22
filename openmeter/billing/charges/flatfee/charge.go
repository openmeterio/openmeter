package flatfee

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

type ChargeBase struct {
	meta.ManagedResource

	Intent Intent `json:"intent"`
	Status Status `json:"status"`

	State State `json:"state"`
}

func (c ChargeBase) Validate() error {
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

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (c ChargeBase) GetChargeID() meta.ChargeID {
	return meta.ChargeID{
		Namespace: c.Namespace,
		ID:        c.ID,
	}
}

func (c ChargeBase) ErrorAttributes() models.Attributes {
	return models.Attributes{
		"charge_id":   c.ID,
		"namespace":   c.Namespace,
		"charge_type": string(meta.ChargeTypeFlatFee),
	}
}

var _ meta.ChargeAccessor = (*Charge)(nil)

type Charge struct {
	ChargeBase

	Realizations Realizations `json:"realizations"`
}

func (c Charge) GetStatus() Status {
	return c.Status
}

func (c Charge) WithStatus(status Status) Charge {
	c.Status = status
	return c
}

func (c Charge) GetBase() ChargeBase {
	return c.ChargeBase
}

func (c Charge) WithBase(base ChargeBase) Charge {
	c.ChargeBase = base
	return c
}

func (c Charge) Validate() error {
	var errs []error

	if err := c.ChargeBase.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge base: %w", err))
	}

	if err := c.Realizations.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("realizations: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
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
}

func (i Intent) Normalized() Intent {
	i.Intent = i.Intent.Normalized()
	i.InvoiceAt = meta.NormalizeTimestamp(i.InvoiceAt)

	calc, err := i.Currency.Calculator()
	if err == nil {
		i.AmountBeforeProration = calc.RoundToPrecision(i.AmountBeforeProration)
	}

	return i
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

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// CalculateAmountAfterProration computes the prorated amount from AmountBeforeProration,
// ServicePeriod, and FullServicePeriod. Returns AmountBeforeProration when proration is
// not applicable (disabled, unsupported mode, or zero-length periods).
func (i Intent) CalculateAmountAfterProration() (alpacadecimal.Decimal, error) {
	if !i.ProRating.Enabled {
		return i.AmountBeforeProration, nil
	}

	if i.ProRating.Mode != productcatalog.ProRatingModeProratePrices {
		return i.AmountBeforeProration, nil
	}

	servicePeriodDuration := int64(i.ServicePeriod.Duration())
	fullServicePeriodDuration := int64(i.FullServicePeriod.Duration())

	// Proration must never increase the amount beyond AmountBeforeProration.
	// Zero-length periods or ServicePeriod >= FullServicePeriod means no proration applies.
	if servicePeriodDuration == 0 || fullServicePeriodDuration == 0 || servicePeriodDuration >= fullServicePeriodDuration {
		return i.AmountBeforeProration, nil
	}

	percentage := alpacadecimal.NewFromInt(servicePeriodDuration).Div(alpacadecimal.NewFromInt(fullServicePeriodDuration))
	amount := i.AmountBeforeProration.Mul(percentage)

	calc, err := i.Currency.Calculator()
	if err != nil {
		return alpacadecimal.Decimal{}, fmt.Errorf("creating currency calculator: %w", err)
	}

	return calc.RoundToPrecision(amount), nil
}

type State struct {
	AdvanceAfter         *time.Time            `json:"advanceAfter,omitempty"`
	FeatureID            *string               `json:"featureId,omitempty"`
	AmountAfterProration alpacadecimal.Decimal `json:"amountAfterProration"`
}

func (s State) Normalized() State {
	s.AdvanceAfter = meta.NormalizeOptionalTimestamp(s.AdvanceAfter)

	return s
}

func (s State) Validate() error {
	var errs []error

	if s.AdvanceAfter != nil {
		if s.AdvanceAfter.IsZero() {
			errs = append(errs, fmt.Errorf("advance after is required"))
		}
	}

	if s.AmountAfterProration.IsNegative() {
		errs = append(errs, fmt.Errorf("amount after proration cannot be negative"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type Realizations struct {
	CreditRealizations creditrealization.Realizations `json:"creditRealizations"`
	AccruedUsage       *invoicedusage.AccruedUsage    `json:"accruedUsage"`
	Payment            *payment.Invoiced              `json:"payment"`
	DetailedLines      mo.Option[DetailedLines]       `json:"detailedLines,omitzero"`
}

func (r Realizations) Validate() error {
	var errs []error

	for _, realization := range r.CreditRealizations {
		if err := realization.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("credit realization[id=%s]: %w", realization.ID, err))
		}
	}

	if r.Payment != nil {
		if err := r.Payment.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("payment[id=%s]: %w", r.Payment.ID, err))
		}
	}

	if r.AccruedUsage != nil {
		if err := r.AccruedUsage.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("accrued usage[id=%s]: %w", r.AccruedUsage.ID, err))
		}
	}

	if r.DetailedLines.IsPresent() {
		if err := r.DetailedLines.OrEmpty().Validate(); err != nil {
			errs = append(errs, fmt.Errorf("detailed lines: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
