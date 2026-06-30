package flatfee

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
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
		ID:        c.Intent.GetCustomerID(),
	}
}

func (c ChargeBase) GetCurrency() currencyx.Code {
	return c.Intent.GetCurrency()
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
	IntentMutableFields `json:"intentMutableFields"`
	SettlementMode      productcatalog.SettlementMode `json:"settlementMode"`
	FeatureKey          *string                       `json:"featureKey,omitempty"`
}

func (i Intent) Normalized() Intent {
	i.IntentMutableFields = i.IntentMutableFields.Normalized(i.Currency)

	return i
}

// AsOverridableIntent maps the intent's mutable fields as the base layer.
func (i Intent) AsOverridableIntent() OverridableIntent {
	return OverridableIntent{
		intent:         i.Intent,
		baseLayer:      i.IntentMutableFields,
		settlementMode: i.SettlementMode,
		featureKey:     i.FeatureKey,
	}
}

func (i Intent) Validate() error {
	var errs []error

	if err := i.Intent.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := i.IntentMutableFields.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := i.SettlementMode.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("settlement mode: %w", err))
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

// OverridableIntent stores the immutable intent plus the base and optional
// override mutable layers. Direct layer access is error-prone because callers
// must manually decide which layer is active; this API centralizes that choice
// so reads and mutations use the correct override layer when present.
type OverridableIntent struct {
	intent meta.Intent

	baseLayer     IntentMutableFields
	overrideLayer *IntentMutableFields

	settlementMode productcatalog.SettlementMode
	featureKey     *string
}

func NewOverridableIntent(baseIntent Intent, overrideLayer *IntentMutableFields) OverridableIntent {
	return OverridableIntent{
		intent:         baseIntent.Intent,
		baseLayer:      baseIntent.IntentMutableFields,
		overrideLayer:  overrideLayer,
		settlementMode: baseIntent.SettlementMode,
		featureKey:     baseIntent.FeatureKey,
	}
}

func (i OverridableIntent) Normalized() OverridableIntent {
	i.baseLayer = i.baseLayer.Normalized(i.intent.Currency)
	if i.overrideLayer != nil {
		overrideLayer := i.overrideLayer.Normalized(i.intent.Currency)
		i.overrideLayer = &overrideLayer
	}

	return i
}

func (i OverridableIntent) GetCustomerID() string {
	return i.intent.CustomerID
}

func (i OverridableIntent) GetCurrency() currencyx.Code {
	return i.intent.Currency
}

func (i OverridableIntent) GetSettlementMode() productcatalog.SettlementMode {
	return i.settlementMode
}

func (i OverridableIntent) GetUniqueReferenceID() *string {
	return i.intent.UniqueReferenceID
}

func (i OverridableIntent) GetSubscription() *meta.SubscriptionReference {
	if i.intent.Subscription == nil {
		return nil
	}

	subscription := *i.intent.Subscription

	return &subscription
}

func (i OverridableIntent) Validate() error {
	var errs []error

	if err := i.intent.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intent: %w", err))
	}

	if err := i.baseLayer.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("base layer: %w", err))
	}

	if i.overrideLayer != nil {
		if err := i.overrideLayer.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("override layer: %w", err))
		}
	}

	if err := i.settlementMode.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("settlement mode: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// GetEffectiveIntent returns the customer-facing intent by combining the
// immutable intent with the active mutable layer.
//
// WARNING: this clones and normalizes the intent and mutable fields. Prefer the
// narrower effective getters when only a few fields are required.
func (i OverridableIntent) GetEffectiveIntent() Intent {
	var featureKey *string
	if i.featureKey != nil {
		featureKey = lo.ToPtr(*i.featureKey)
	}

	intent := Intent{
		Intent:              i.intent.Clone(),
		IntentMutableFields: i.baseLayer.Clone(),
		SettlementMode:      i.settlementMode,
		FeatureKey:          featureKey,
	}

	if i.overrideLayer != nil {
		intent.IntentMutableFields = i.overrideLayer.Clone()
	}

	return intent.Normalized()
}

// GetEffectiveServicePeriod returns the service period from the active mutable
// layer, preferring the override layer when it is present.
func (i OverridableIntent) GetEffectiveServicePeriod() timeutil.ClosedPeriod {
	if i.overrideLayer != nil {
		return i.overrideLayer.ServicePeriod
	}

	return i.baseLayer.ServicePeriod
}

// GetEffectiveInvoiceAt returns the invoice-at timestamp from the active
// mutable layer, preferring the override layer when it is present.
func (i OverridableIntent) GetEffectiveInvoiceAt() time.Time {
	if i.overrideLayer != nil {
		return i.overrideLayer.InvoiceAt
	}

	return i.baseLayer.InvoiceAt
}

// GetEffectivePaymentTerm returns the payment term from the active mutable
// layer, preferring the override layer when it is present.
func (i OverridableIntent) GetEffectivePaymentTerm() productcatalog.PaymentTermType {
	if i.overrideLayer != nil {
		return i.overrideLayer.PaymentTerm
	}

	return i.baseLayer.PaymentTerm
}

// GetFeatureKey returns the immutable flat-fee feature key from the
// base intent. Override layers cannot change feature attribution.
func (i OverridableIntent) GetFeatureKey() string {
	if i.featureKey == nil {
		return ""
	}

	return *i.featureKey
}

// GetTaxConfig returns the immutable tax config from the base intent.
// Override layers cannot change tax attribution.
func (i OverridableIntent) GetTaxConfig() productcatalog.TaxCodeConfig {
	return i.intent.TaxConfig
}

// GetEffectiveMetaIntentMutableFields returns the shared meta mutable fields
// from the active mutable layer, preferring the override layer when it is
// present.
func (i OverridableIntent) GetEffectiveMetaIntentMutableFields() meta.IntentMutableFields {
	if i.overrideLayer != nil {
		return i.overrideLayer.IntentMutableFields
	}

	return i.baseLayer.IntentMutableFields
}

func (i OverridableIntent) GetBaseManagedBy() billing.InvoiceLineManagedBy {
	return i.intent.ManagedBy
}

func (i OverridableIntent) GetBaseIntent() Intent {
	var featureKey *string
	if i.featureKey != nil {
		featureKey = lo.ToPtr(*i.featureKey)
	}

	return Intent{
		Intent:              i.intent.Clone(),
		IntentMutableFields: i.baseLayer.Clone(),
		SettlementMode:      i.settlementMode,
		FeatureKey:          featureKey,
	}
}

func (i OverridableIntent) GetIntentForTarget(target meta.ChangeTarget) (Intent, error) {
	var featureKey *string
	if i.featureKey != nil {
		featureKey = lo.ToPtr(*i.featureKey)
	}

	out := Intent{
		Intent:         i.intent.Clone(),
		SettlementMode: i.settlementMode,
		FeatureKey:     featureKey,
	}

	switch target {
	case meta.ChangeTargetBase:
		out.IntentMutableFields = i.baseLayer.Clone()
	case meta.ChangeTargetOverride:
		if i.overrideLayer == nil {
			return Intent{}, fmt.Errorf("override layer not present for charge")
		}

		out.IntentMutableFields = i.overrideLayer.Clone()
	default:
		return Intent{}, fmt.Errorf("invalid change target: %s", target)
	}

	return out, nil
}

func (i OverridableIntent) GetOverrideLayerMutableFields() *IntentMutableFields {
	if i.overrideLayer == nil {
		return nil
	}

	return lo.ToPtr(i.overrideLayer.Clone())
}

func (i OverridableIntent) HasOverrideLayer() bool {
	return i.overrideLayer != nil
}

func (i OverridableIntent) GetDeletedAt() *time.Time {
	if i.overrideLayer != nil {
		return i.overrideLayer.IntentDeletedAt
	}

	return i.baseLayer.IntentDeletedAt
}

func (i OverridableIntent) CalculateAmountAfterProration() (alpacadecimal.Decimal, error) {
	// TODO[later,performance]: We should not clone for this, but this is not on a hot path.
	return i.GetEffectiveIntent().CalculateAmountAfterProration()
}

func (i *OverridableIntent) MutateEffective(editFn func(*IntentMutableFields)) error {
	target := meta.ChangeTargetBase
	if i.overrideLayer != nil {
		target = meta.ChangeTargetOverride
	}

	return i.Mutate(target, editFn)
}

// Mutate edits the requested intent mutable field layer.
//
// The callback always receives a non-nil pointer to a cloned mutable-field value.
// The clone is written back only after it normalizes and validates, so validation
// errors do not partially mutate the intent.
func (i *OverridableIntent) Mutate(target meta.ChangeTarget, editFn func(*IntentMutableFields)) error {
	var targetFields IntentMutableFields
	switch target {
	case meta.ChangeTargetBase:
		targetFields = i.baseLayer.Clone()
	case meta.ChangeTargetOverride:
		if i.overrideLayer == nil {
			return fmt.Errorf("override layer not present for charge")
		}

		targetFields = i.overrideLayer.Clone()
	}

	editFn(&targetFields)

	normalizedFields := targetFields.Normalized(i.intent.Currency)
	if err := normalizedFields.Validate(); err != nil {
		return fmt.Errorf("validating intent: %w", err)
	}

	switch target {
	case meta.ChangeTargetBase:
		i.baseLayer = normalizedFields
	case meta.ChangeTargetOverride:
		if i.overrideLayer == nil {
			return fmt.Errorf("override layer not present for charge")
		}

		i.overrideLayer = &normalizedFields
	default:
		return fmt.Errorf("invalid change target: %s", target)
	}

	return nil
}

type IntentMutableFields struct {
	meta.IntentMutableFields

	// IntentDeletedAt marks the flat-fee base/original intent as deleted.
	// Adapters derive the effective charge DeletedAt from this value when no intent override is present.
	IntentDeletedAt *time.Time `json:"intentDeletedAt,omitempty"`

	InvoiceAt           time.Time                      `json:"invoiceAt"`
	PaymentTerm         productcatalog.PaymentTermType `json:"paymentTerm"`
	PercentageDiscounts *billing.PercentageDiscount    `json:"percentageDiscounts"`

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

func (f IntentMutableFields) Clone() IntentMutableFields {
	out := f
	out.IntentMutableFields = f.IntentMutableFields.Clone()

	if f.PercentageDiscounts != nil {
		out.PercentageDiscounts = lo.ToPtr(f.PercentageDiscounts.Clone())
	}

	return out
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
