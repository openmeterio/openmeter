package usagebased

import (
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/ref"
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
		ID:        c.Intent.CustomerID,
	}
}

func (c Charge) GetFeatureKeyOrID() ref.IDOrKey {
	switch c.Status {
	case StatusCreated:
		return ref.IDOrKey{
			Key: c.Intent.BaseLayer.FeatureKey,
		}
	case StatusDeleted:
		if c.State.FeatureID != "" {
			return ref.IDOrKey{
				ID: c.State.FeatureID,
			}
		}

		return ref.IDOrKey{
			Key: c.Intent.BaseLayer.FeatureKey,
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
		Intent:         i.Intent,
		BaseLayer:      i.IntentMutableFields,
		SettlementMode: i.SettlementMode,
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

type OverridableIntent struct {
	meta.Intent

	BaseLayer     IntentMutableFields  `json:"baseLayer"`
	OverrideLayer *IntentMutableFields `json:"overrideLayer,omitempty"`

	SettlementMode productcatalog.SettlementMode `json:"settlementMode"`
}

func (i OverridableIntent) Normalized() OverridableIntent {
	i.BaseLayer = i.BaseLayer.Normalized()
	if i.OverrideLayer != nil {
		overrideLayer := i.OverrideLayer.Normalized()
		i.OverrideLayer = &overrideLayer
	}

	return i
}

func (i OverridableIntent) Validate() error {
	var errs []error

	if err := i.Intent.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("intent: %w", err))
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

func (i OverridableIntent) GetEffectiveIntent() Intent {
	intent := Intent{
		Intent:              i.Intent.Clone(),
		IntentMutableFields: i.BaseLayer.Clone(),
		SettlementMode:      i.SettlementMode,
	}

	if i.OverrideLayer != nil {
		intent.IntentMutableFields = i.OverrideLayer.Clone()
	}

	return intent.Normalized()
}

func (i *OverridableIntent) Mutate(target meta.ChangeTarget) (MutableIntent, error) {
	return newMutableIntent(i, target)
}

type MutableIntent interface {
	SetServicePeriodTo(time.Time) MutableIntent
	Save() error
}

var _ MutableIntent = (*mutableIntent)(nil)

type mutableIntent struct {
	intent        *OverridableIntent
	mutableFields IntentMutableFields
	target        meta.ChangeTarget
}

func newMutableIntent(intent *OverridableIntent, target meta.ChangeTarget) (*mutableIntent, error) {
	switch target {
	case meta.ChangeTargetBase:
		return &mutableIntent{
			intent:        intent,
			mutableFields: intent.BaseLayer.Clone(),
			target:        target,
		}, nil
	case meta.ChangeTargetOverride:
		if intent.OverrideLayer == nil {
			return nil, fmt.Errorf("override layer not present for charge")
		}

		return &mutableIntent{
			intent:        intent,
			mutableFields: intent.OverrideLayer.Clone(),
			target:        target,
		}, nil
	default:
		return nil, fmt.Errorf("invalid change target: %s", target)
	}
}

func (m *mutableIntent) SetServicePeriodTo(to time.Time) MutableIntent {
	m.mutableFields.ServicePeriod.To = to
	return m
}

func (m *mutableIntent) Save() error {
	normalizedFields := m.mutableFields.Normalized()

	if err := normalizedFields.Validate(); err != nil {
		return fmt.Errorf("validating intent: %w", err)
	}

	switch m.target {
	case meta.ChangeTargetBase:
		m.intent.BaseLayer = normalizedFields
	case meta.ChangeTargetOverride:
		m.intent.OverrideLayer = &normalizedFields
	default:
		return fmt.Errorf("invalid change target: %s", m.target)
	}

	m.mutableFields = normalizedFields

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

	Discounts productcatalog.Discounts `json:"discounts"`
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
