package flatfee

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type ChargeBase struct {
	meta.ManagedResource

	Intent OverridableIntent `json:"intent"`
	Status Status            `json:"status"`

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

func (c ChargeBase) GetCustomerID() customer.CustomerID {
	return customer.CustomerID{
		Namespace: c.Namespace,
		ID:        c.Intent.CustomerID,
	}
}

func (c ChargeBase) GetCurrency() currencyx.Code {
	return c.Intent.Currency
}

// GetIntentDeletedAt returns the effective intent deletion timestamp.
// If an override is present, the override intent owns deletion; otherwise the base intent does.
func (c ChargeBase) GetIntentDeletedAt() *time.Time {
	if c.Intent.OverrideLayer != nil {
		return c.Intent.OverrideLayer.IntentDeletedAt
	}

	return c.Intent.BaseLayer.IntentDeletedAt
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

type OverridableIntent struct {
	meta.Intent

	BaseLayer     IntentMutableFields  `json:"baseLayer"`
	OverrideLayer *IntentMutableFields `json:"overrideLayer,omitempty"`

	SettlementMode productcatalog.SettlementMode `json:"settlementMode"`
}

type Intent struct {
	meta.Intent
	IntentMutableFields `json:"intentMutableFields"`
	SettlementMode      productcatalog.SettlementMode `json:"settlementMode"`
}

func (i Intent) AsOverridableIntent() OverridableIntent {
	return OverridableIntent{
		Intent:         i.Intent,
		BaseLayer:      i.IntentMutableFields,
		SettlementMode: i.SettlementMode,
	}
}

func (i Intent) Normalized() Intent {
	i.IntentMutableFields = i.IntentMutableFields.Normalized(i.Currency)

	return i
}

func (i Intent) Validate() error {
	return i.AsOverridableIntent().Validate()
}

type IntentMutableFields struct {
	meta.IntentMutableFields

	// IntentDeletedAt marks the flat-fee base/original intent as deleted.
	// Adapters derive the effective charge DeletedAt from this value when no intent override is present.
	IntentDeletedAt *time.Time `json:"intentDeletedAt,omitempty"`

	InvoiceAt           time.Time                          `json:"invoiceAt"`
	PaymentTerm         productcatalog.PaymentTermType     `json:"paymentTerm"`
	FeatureKey          string                             `json:"featureKey,omitempty"`
	PercentageDiscounts *productcatalog.PercentageDiscount `json:"percentageDiscounts"`

	ProRating             productcatalog.ProRatingConfig `json:"proRating"`
	AmountBeforeProration alpacadecimal.Decimal          `json:"amountBeforeProration"`
}

func (f IntentMutableFields) Normalized(currency currencyx.Code) IntentMutableFields {
	f.IntentMutableFields = f.IntentMutableFields.Normalized()
	f.InvoiceAt = meta.NormalizeTimestamp(f.InvoiceAt)

	calc, err := currency.Calculator()
	if err == nil {
		f.AmountBeforeProration = calc.RoundToPrecision(f.AmountBeforeProration)
	}

	return f
}

func (f IntentMutableFields) Validate() error {
	var errs []error

	if err := f.IntentMutableFields.Validate(); err != nil {
		errs = append(errs, err)
	}

	if f.AmountBeforeProration.IsNegative() {
		errs = append(errs, fmt.Errorf("amount before proration cannot be negative"))
	}

	if !slices.Contains(productcatalog.PaymentTermType("").Values(), string(f.PaymentTerm)) {
		errs = append(errs, fmt.Errorf("invalid payment term %s", f.PaymentTerm))
	}

	if f.InvoiceAt.IsZero() {
		errs = append(errs, fmt.Errorf("invoice at is required"))
	}

	if f.PercentageDiscounts != nil {
		if err := f.PercentageDiscounts.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("percentage discounts: %w", err))
		}
	}

	if err := f.ProRating.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("pro rating: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (i OverridableIntent) Normalized() OverridableIntent {
	i.BaseLayer = i.BaseLayer.Normalized(i.Currency)
	if i.OverrideLayer != nil {
		overrideLayer := i.OverrideLayer.Normalized(i.Currency)
		i.OverrideLayer = &overrideLayer
	}

	return i
}

func (i OverridableIntent) Validate() error {
	var errs []error

	if err := i.Intent.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := i.BaseLayer.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("base layer: %w", err))
	}

	if i.OverrideLayer != nil {
		if err := i.OverrideLayer.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("override layer: %w", err))
		}
	}

	if err := i.SettlementMode.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("settlement mode: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (i OverridableIntent) EffectiveIntent() Intent {
	intent := Intent{
		Intent:              i.Intent,
		IntentMutableFields: i.BaseLayer,
		SettlementMode:      i.SettlementMode,
	}

	if i.OverrideLayer != nil {
		intent.IntentMutableFields = *i.OverrideLayer
	}

	return intent.Normalized()
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

func (i OverridableIntent) CalculateAmountAfterProration() (alpacadecimal.Decimal, error) {
	return i.EffectiveIntent().CalculateAmountAfterProration()
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
	CurrentRun *RealizationRun `json:"currentRun,omitempty"`
	PriorRuns  RealizationRuns `json:"priorRuns,omitempty"`
}

func (r Realizations) GetByLineID(lineID string) (RealizationRun, error) {
	if r.CurrentRun != nil {
		if r.CurrentRun.LineID != nil && *r.CurrentRun.LineID == lineID {
			return *r.CurrentRun, nil
		}
	}

	for _, run := range r.PriorRuns {
		if run.LineID != nil && *run.LineID == lineID {
			return run, nil
		}
	}

	return RealizationRun{}, fmt.Errorf("realization run not found [line_id=%s]", lineID)
}

func (r Realizations) Validate() error {
	var errs []error

	if r.CurrentRun != nil {
		if err := r.CurrentRun.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("current run: %w", err))
		}
	}

	if err := r.PriorRuns.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("prior runs: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
