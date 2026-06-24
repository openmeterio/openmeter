package flatfee

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type ChargeBase struct {
	meta.ManagedResource

	Intent         Intent          `json:"intent"`
	IntentOverride *IntentOverride `json:"intentOverride,omitempty"`
	Status         Status          `json:"status"`

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

	if c.IntentOverride != nil {
		if err := c.IntentOverride.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("intent override: %w", err))
		}
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
	if c.IntentOverride != nil {
		return c.IntentOverride.IntentDeletedAt
	}

	return c.Intent.IntentDeletedAt
}

func (c ChargeBase) GetMergedIntent() Intent {
	intent := c.Intent
	if c.IntentOverride == nil {
		return intent
	}

	override := c.IntentOverride
	intent.Name = override.Name
	intent.Description = override.Description
	intent.Metadata = override.Metadata
	intent.TaxConfig = productcatalog.TaxCodeConfig{
		Behavior:  override.TaxBehavior,
		TaxCodeID: lo.FromPtr(override.TaxCodeID),
	}
	intent.IntentDeletedAt = override.IntentDeletedAt
	intent.ServicePeriod = override.ServicePeriod
	intent.FullServicePeriod = override.FullServicePeriod
	intent.BillingPeriod = override.BillingPeriod
	intent.InvoiceAt = override.InvoiceAt
	intent.FeatureKey = override.FeatureKey
	intent.PaymentTerm = override.PaymentTerm
	intent.ProRating = override.ProRating
	intent.AmountBeforeProration = override.AmountBeforeProration
	intent.PercentageDiscounts = override.PercentageDiscounts

	return intent
}

func (c *ChargeBase) SetMergedIntent(intent Intent) {
	if c.IntentOverride == nil {
		c.Intent = intent
		return
	}

	c.IntentOverride.Name = intent.Name
	c.IntentOverride.Description = intent.Description
	c.IntentOverride.Metadata = intent.Metadata
	c.IntentOverride.TaxBehavior = intent.TaxConfig.Behavior
	c.IntentOverride.TaxCodeID = lo.EmptyableToPtr(intent.TaxConfig.TaxCodeID)
	c.IntentOverride.IntentDeletedAt = intent.IntentDeletedAt
	c.IntentOverride.ServicePeriod = intent.ServicePeriod
	c.IntentOverride.FullServicePeriod = intent.FullServicePeriod
	c.IntentOverride.BillingPeriod = intent.BillingPeriod
	c.IntentOverride.InvoiceAt = intent.InvoiceAt
	c.IntentOverride.FeatureKey = intent.FeatureKey
	c.IntentOverride.PaymentTerm = intent.PaymentTerm
	c.IntentOverride.ProRating = intent.ProRating
	c.IntentOverride.AmountBeforeProration = intent.AmountBeforeProration
	c.IntentOverride.PercentageDiscounts = intent.PercentageDiscounts
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

	// IntentDeletedAt marks the flat-fee base/original intent as deleted.
	// Adapters derive the effective charge DeletedAt from this value when no intent override is present.
	IntentDeletedAt *time.Time `json:"intentDeletedAt,omitempty"`

	PaymentTerm         productcatalog.PaymentTermType     `json:"paymentTerm"`
	FeatureKey          string                             `json:"featureKey,omitempty"`
	PercentageDiscounts *productcatalog.PercentageDiscount `json:"percentageDiscounts"`

	ProRating             productcatalog.ProRatingConfig `json:"proRating"`
	AmountBeforeProration alpacadecimal.Decimal          `json:"amountBeforeProration"`
}

type IntentOverride struct {
	Name        string          `json:"name"`
	Description *string         `json:"description,omitempty"`
	Metadata    models.Metadata `json:"metadata,omitempty"`

	TaxBehavior *productcatalog.TaxBehavior `json:"taxBehavior,omitempty"`
	TaxCodeID   *string                     `json:"taxCodeID,omitempty"`

	// IntentDeletedAt marks the flat-fee override intent as deleted.
	// When an override is present, adapters derive the effective charge DeletedAt from this value instead of the base intent.
	IntentDeletedAt *time.Time `json:"intentDeletedAt,omitempty"`

	ServicePeriod     timeutil.ClosedPeriod `json:"servicePeriod"`
	FullServicePeriod timeutil.ClosedPeriod `json:"fullServicePeriod"`
	BillingPeriod     timeutil.ClosedPeriod `json:"billingPeriod"`
	InvoiceAt         time.Time             `json:"invoiceAt"`

	FeatureKey            string                             `json:"featureKey,omitempty"`
	PaymentTerm           productcatalog.PaymentTermType     `json:"paymentTerm"`
	ProRating             productcatalog.ProRatingConfig     `json:"proRating"`
	AmountBeforeProration alpacadecimal.Decimal              `json:"amountBeforeProration"`
	PercentageDiscounts   *productcatalog.PercentageDiscount `json:"percentageDiscounts,omitempty"`
}

func (o IntentOverride) Normalized() IntentOverride {
	o.ServicePeriod = meta.NormalizeClosedPeriod(o.ServicePeriod)
	o.FullServicePeriod = meta.NormalizeClosedPeriod(o.FullServicePeriod)
	o.BillingPeriod = meta.NormalizeClosedPeriod(o.BillingPeriod)
	o.InvoiceAt = meta.NormalizeTimestamp(o.InvoiceAt)

	return o
}

func (o IntentOverride) Validate() error {
	var errs []error

	if o.Name == "" {
		errs = append(errs, errors.New("name cannot be empty"))
	}

	if o.TaxBehavior != nil {
		if err := o.TaxBehavior.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("tax behavior: %w", err))
		}
	}

	if err := o.ServicePeriod.ValidateAsRequired(); err != nil {
		errs = append(errs, fmt.Errorf("service period: %w", err))
	}

	if err := o.FullServicePeriod.ValidateAsRequired(); err != nil {
		errs = append(errs, fmt.Errorf("full service period: %w", err))
	}

	if err := o.BillingPeriod.ValidateAsRequired(); err != nil {
		errs = append(errs, fmt.Errorf("billing period: %w", err))
	}

	if o.InvoiceAt.IsZero() {
		errs = append(errs, errors.New("invoice at is required"))
	}

	if err := o.PaymentTerm.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("payment term: %w", err))
	}

	if err := o.ProRating.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("pro rating: %w", err))
	}

	if o.AmountBeforeProration.IsNegative() {
		errs = append(errs, errors.New("amount before proration cannot be negative"))
	}

	if o.PercentageDiscounts != nil {
		if err := o.PercentageDiscounts.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("percentage discounts: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
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
