package usagebased

import (
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/ref"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

var _ meta.ChargeAccessor = (*ChargeBase)(nil)

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

// GetIntentDeletedAt returns the effective intent deletion timestamp.
// If an override is present, the override intent owns deletion; otherwise the base intent does.
func (c ChargeBase) GetIntentDeletedAt() *time.Time {
	return c.Intent.GetDeletedAt()
}

func (c ChargeBase) ErrorAttributes() models.Attributes {
	return models.Attributes{
		"charge_id":   c.ID,
		"namespace":   c.Namespace,
		"charge_type": string(meta.ChargeTypeUsageBased),
	}
}

var _ meta.ChargeAccessor = (*Charge)(nil)

type Charge struct {
	ChargeBase

	Realizations RealizationRuns `json:"realizations"`
	Expands      Expands         `json:"expands"`
}

type Charges []Charge

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

	if err := c.Expands.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("expands: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (c Charge) GetCurrentRealizationRun() (RealizationRun, error) {
	if c.State.CurrentRealizationRunID == nil {
		return RealizationRun{}, fmt.Errorf("no current realization run")
	}

	return c.Realizations.GetByID(*c.State.CurrentRealizationRunID)
}

func (c Charge) GetCustomerID() customer.CustomerID {
	return customer.CustomerID{
		Namespace: c.Namespace,
		ID:        c.Intent.GetCustomerID(),
	}
}

func (c Charge) GetFeatureKeyOrID() ref.IDOrKey {
	// TODO: if API edits can override FeatureKey, re-resolve State.FeatureID
	// whenever the effective key changes. Call chain: API line edit -> usage-based
	// line engine -> charge override -> feature resolution/current totals/triggers.
	// State.FeatureID is the persisted resolved feature snapshot used by active
	// charges; created/deleted fallbacks resolve by key.
	switch c.Status {
	case StatusCreated:
		return ref.IDOrKey{
			Key: c.Intent.GetBaseIntent().FeatureKey,
		}
	case StatusDeleted:
		if c.State.FeatureID != "" {
			return ref.IDOrKey{
				ID: c.State.FeatureID,
			}
		}

		return ref.IDOrKey{
			Key: c.Intent.GetBaseIntent().FeatureKey,
		}
	default:
		return ref.IDOrKey{
			ID: c.State.FeatureID,
		}
	}
}

func (c Charge) ResolveFeatureMeter(featureMeters feature.FeatureMeters) (feature.FeatureMeter, error) {
	const requireMeter = true

	featureRef := c.GetFeatureKeyOrID()
	if featureRef.ID != "" {
		return featureMeters.GetByID(featureRef.ID, requireMeter)
	}

	featureMeter, err := featureMeters.Get(featureRef.Key, requireMeter)
	if err != nil {
		return feature.FeatureMeter{}, fmt.Errorf("get feature meter: %w", err)
	}

	return featureMeter, nil
}

// GetFeatureKeysOrIDs returns the unique state-aware feature references for the charges.
// Each charge contributes the ref returned by GetFeatureKeyOrID, so created charges use keys,
// deleted charges prefer IDs and fall back to keys, and all other states use IDs.
func (c Charges) GetFeatureKeysOrIDs() []ref.IDOrKey {
	return lo.Uniq(lo.Map(c, func(charge Charge, _ int) ref.IDOrKey {
		return charge.GetFeatureKeyOrID()
	}))
}

type Intent struct {
	meta.Intent
	IntentMutableFields `json:"intentMutableFields"`
	SettlementMode      productcatalog.SettlementMode `json:"settlementMode"`
}

// AsOverridableIntent maps the intent's mutable fields as the base layer.
func (i Intent) AsOverridableIntent() OverridableIntent {
	return OverridableIntent{
		intent:         i.Intent,
		baseLayer:      i.IntentMutableFields,
		settlementMode: i.SettlementMode,
	}
}

func (i Intent) Normalized() Intent {
	i.IntentMutableFields = i.IntentMutableFields.Normalized()

	return i
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

// OverridableIntent stores the immutable intent plus the base and optional
// override mutable layers. Direct layer access is error-prone because callers
// must manually decide which layer is active; this API centralizes that choice
// so reads and mutations use the correct override layer when present.
type OverridableIntent struct {
	intent meta.Intent

	baseLayer     IntentMutableFields
	overrideLayer *IntentMutableFields

	settlementMode productcatalog.SettlementMode
}

func NewOverridableIntent(baseIntent Intent, overrideLayer *IntentMutableFields) OverridableIntent {
	return OverridableIntent{
		intent:         baseIntent.Intent,
		baseLayer:      baseIntent.IntentMutableFields,
		overrideLayer:  overrideLayer,
		settlementMode: baseIntent.SettlementMode,
	}
}

func (i OverridableIntent) Normalized() OverridableIntent {
	i.baseLayer = i.baseLayer.Normalized()
	if i.overrideLayer != nil {
		overrideLayer := i.overrideLayer.Normalized()
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
	intent := Intent{
		Intent:              i.intent.Clone(),
		IntentMutableFields: i.baseLayer.Clone(),
		SettlementMode:      i.settlementMode,
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

// GetEffectiveFeatureKey returns the feature key from the active mutable layer,
// preferring the override layer when it is present.
func (i OverridableIntent) GetEffectiveFeatureKey() string {
	if i.overrideLayer != nil {
		return i.overrideLayer.FeatureKey
	}

	return i.baseLayer.FeatureKey
}

// GetEffectivePrice returns a cloned price from the active mutable layer,
// preferring the override layer when it is present.
func (i OverridableIntent) GetEffectivePrice() productcatalog.Price {
	if i.overrideLayer != nil {
		return *i.overrideLayer.Price.Clone()
	}

	return *i.baseLayer.Price.Clone()
}

// GetEffectiveDiscounts returns cloned discounts from the active mutable layer,
// preferring the override layer when it is present.
func (i OverridableIntent) GetEffectiveDiscounts() billing.Discounts {
	if i.overrideLayer != nil {
		return i.overrideLayer.Discounts.Clone()
	}

	return i.baseLayer.Discounts.Clone()
}

// GetEffectiveTaxConfig returns the tax config from the active mutable layer,
// preferring the override layer when it is present.
func (i OverridableIntent) GetEffectiveTaxConfig() productcatalog.TaxCodeConfig {
	if i.overrideLayer != nil {
		return i.overrideLayer.TaxConfig
	}

	return i.baseLayer.TaxConfig
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

// GetBaseTaxConfig returns the tax config from the base mutable layer,
// ignoring any override layer.
func (i OverridableIntent) GetBaseTaxConfig() productcatalog.TaxCodeConfig {
	return i.baseLayer.TaxConfig
}

func (i OverridableIntent) GetBaseIntent() Intent {
	return Intent{
		Intent:              i.intent.Clone(),
		IntentMutableFields: i.baseLayer.Clone(),
		SettlementMode:      i.settlementMode,
	}
}

func (i OverridableIntent) GetIntentForTarget(target meta.ChangeTarget) (Intent, error) {
	out := Intent{
		Intent:         i.intent.Clone(),
		SettlementMode: i.settlementMode,
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

	normalizedFields := targetFields.Normalized()

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

	// IntentDeletedAt marks the usage-based base/original intent as deleted.
	// Adapters derive the effective charge DeletedAt from this value when no intent override is present.
	IntentDeletedAt *time.Time `json:"intentDeletedAt,omitempty"`

	InvoiceAt time.Time `json:"invoiceAt"`

	FeatureKey string `json:"featureKey"`

	Price productcatalog.Price `json:"price"`

	Discounts billing.Discounts `json:"discounts"`

	// UnitConfig is the optional unit conversion snapshotted from the effective
	// rate card. Like Price it is a mutable rating input (set on create and
	// update) so re-rates read the config in effect for the charge.
	UnitConfig *productcatalog.UnitConfig `json:"unitConfig,omitempty"`
}

func (f IntentMutableFields) Normalized() IntentMutableFields {
	f.IntentMutableFields = f.IntentMutableFields.Normalized()
	f.InvoiceAt = meta.NormalizeTimestamp(f.InvoiceAt)

	return f
}

func (f IntentMutableFields) Clone() IntentMutableFields {
	out := f
	out.IntentMutableFields = f.IntentMutableFields.Clone()
	out.Price = *f.Price.Clone()
	out.Discounts = f.Discounts.Clone()

	if f.UnitConfig != nil {
		out.UnitConfig = lo.ToPtr(f.UnitConfig.Clone())
	}

	return out
}

func (f IntentMutableFields) Validate() error {
	var errs []error

	if err := f.IntentMutableFields.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intent mutable fields: %w", err))
	}

	if err := f.Discounts.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("discounts: %w", err))
	}

	if f.InvoiceAt.IsZero() {
		errs = append(errs, fmt.Errorf("invoice at is required"))
	}

	if err := f.Price.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("price: %w", err))
	}

	if err := f.UnitConfig.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("unit config: %w", err))
	}

	if f.FeatureKey == "" {
		errs = append(errs, fmt.Errorf("feature key is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type State struct {
	CurrentRealizationRunID *string      `json:"currentRealizationRunId"`
	AdvanceAfter            *time.Time   `json:"advanceAfter"`
	FeatureID               string       `json:"featureId"`
	RatingEngine            RatingEngine `json:"ratingEngine"`
}

func (s State) Normalized() State {
	s.AdvanceAfter = meta.NormalizeOptionalTimestamp(s.AdvanceAfter)

	return s
}

func (s State) Validate() error {
	var errs []error

	if s.CurrentRealizationRunID != nil && *s.CurrentRealizationRunID == "" {
		errs = append(errs, fmt.Errorf("current realization run ID must be non-empty"))
	}

	if s.FeatureID == "" {
		errs = append(errs, fmt.Errorf("feature id must be set"))
	}

	if err := s.RatingEngine.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("rating engine: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type Expands struct {
	RealtimeUsage *totals.Totals `json:"realtimeUsage,omitempty"`
}

func (e Expands) Validate() error {
	var errs []error

	if e.RealtimeUsage != nil {
		if err := e.RealtimeUsage.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("realtime usage: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
