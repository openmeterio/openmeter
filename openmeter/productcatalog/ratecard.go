package productcatalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

const (
	FlatFeeRateCardType    RateCardType = "flat_fee"
	UsageBasedRateCardType RateCardType = "usage_based"
)

type RateCardType string

func (s RateCardType) Values() []string {
	return []string{
		string(FlatFeeRateCardType),
		string(UsageBasedRateCardType),
	}
}

type RateCard interface {
	models.Validator
	models.Equaler[RateCard]

	Type() RateCardType
	AsMeta() RateCardMeta
	Key() string
	Merge(RateCard) error
	ChangeMeta(func(m RateCardMeta) (RateCardMeta, error)) error
	Clone() RateCard
	Compatible(RateCard) error
	GetBillingCadence() *datetime.ISODuration
	IsBillable() bool
}

type RateCardSerde struct {
	Type RateCardType `json:"type"`
}

var (
	_ models.Validator                     = (*RateCardMeta)(nil)
	_ models.Equaler[RateCardMeta]         = (*RateCardMeta)(nil)
	_ models.CustomValidator[RateCardMeta] = (*RateCardMeta)(nil)
)

type RateCardMeta struct {
	// Key is the unique key for Plan.
	Key string `json:"key"`

	// Name of the RateCard
	Name string `json:"name"`

	// Description for the RateCard
	Description *string `json:"description,omitempty"`

	// Metadata a set of key/value pairs describing metadata for the RateCard
	Metadata models.Metadata `json:"metadata,omitempty"`

	// FeatureKey is the key of the feature assigned to the RateCard
	FeatureKey *string `json:"featureKey,omitempty"`

	// FeatureID is the ID of the feature assigned to the RateCard
	FeatureID *string `json:"featureID,omitempty"`

	// EntitlementTemplate defines the template used for instantiating entitlement.Entitlement.
	// If Feature is set then template must be provided as well.
	EntitlementTemplate *EntitlementTemplate `json:"entitlementTemplate,omitempty"`

	// TaxConfig defines provider specific tax information.
	TaxConfig *TaxConfig `json:"taxConfig,omitempty"`

	// Price defines the price for the RateCard
	Price *Price `json:"price"`

	// Discounts defines a list of discounts for the RateCard
	Discounts Discounts `json:"discounts,omitempty"`
}

func (r RateCardMeta) Clone() RateCardMeta {
	clone := RateCardMeta{
		Key:  r.Key,
		Name: r.Name,
	}

	if r.Description != nil {
		desc := *r.Description
		clone.Description = &desc
	}

	// Deep copy metadata map
	if len(r.Metadata) > 0 {
		clone.Metadata = make(map[string]string, len(r.Metadata))
		for k, v := range r.Metadata {
			clone.Metadata[k] = v
		}
	}

	if r.FeatureKey != nil {
		key := *r.FeatureKey
		clone.FeatureKey = &key
	}

	if r.FeatureID != nil {
		id := *r.FeatureID
		clone.FeatureID = &id
	}

	if r.EntitlementTemplate != nil {
		entTmp := *r.EntitlementTemplate
		clone.EntitlementTemplate = &entTmp
	}

	if r.TaxConfig != nil {
		taxCfg := *r.TaxConfig
		clone.TaxConfig = &taxCfg
	}

	if r.Price != nil {
		p := *r.Price.Clone()
		clone.Price = &p
	}

	clone.Discounts = r.Discounts.Clone()

	return clone
}

func (r RateCardMeta) Equal(v RateCardMeta) bool {
	if r.Key != v.Key {
		return false
	}

	if r.Name != v.Name {
		return false
	}

	if lo.FromPtr(r.Description) != lo.FromPtr(v.Description) {
		return false
	}

	if lo.FromPtr(r.FeatureKey) != lo.FromPtr(v.FeatureKey) {
		return false
	}

	if lo.FromPtr(r.FeatureID) != lo.FromPtr(v.FeatureID) {
		return false
	}

	if !r.EntitlementTemplate.Equal(v.EntitlementTemplate) {
		return false
	}

	if !r.TaxConfig.Equal(v.TaxConfig) {
		return false
	}

	if (r.Price != nil && v.Price == nil) ||
		(r.Price == nil && v.Price != nil) {
		return false
	}

	if !r.Discounts.Equal(v.Discounts) {
		return false
	}

	return r.Price.Equal(v.Price)
}

func (r RateCardMeta) ValidateWith(v ...models.ValidatorFunc[RateCardMeta]) error {
	return models.Validate(r, v...)
}

func (r RateCardMeta) Validate() error {
	var errs []error

	if r.EntitlementTemplate != nil {
		if r.FeatureKey == nil {
			errs = append(errs, ErrRateCardEntitlementTemplateWithNoFeature)
		}

		if err := r.EntitlementTemplate.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid entitlement template: %w", err))
		}
	}

	if r.TaxConfig != nil {
		if err := r.TaxConfig.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid tax config: %w",
				models.ErrorWithFieldPrefix(
					models.NewFieldSelectorGroup(models.NewFieldSelector("taxConfig")),
					err),
			))
		}
	}

	if r.Price != nil {
		if err := r.Price.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("invalid price: %w",
				models.ErrorWithFieldPrefix(
					models.NewFieldSelectorGroup(models.NewFieldSelector("price")),
					err),
			))
		}
	}

	if r.FeatureKey != nil {
		if r.Key != *r.FeatureKey {
			errs = append(errs, ErrRateCardKeyFeatureKeyMismatch)
		}
	}

	if err := r.Discounts.ValidateForPrice(r.Price); err != nil {
		errs = append(errs, err)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (r RateCardMeta) IsBillable() bool {
	return r.Price != nil
}

var (
	_ RateCard                                 = (*FlatFeeRateCard)(nil)
	_ models.CustomValidator[*FlatFeeRateCard] = (*FlatFeeRateCard)(nil)
)

type FlatFeeRateCard struct {
	RateCardMeta

	// BillingCadence defines the billing cadence of the RateCard in ISO8601 format.
	// When nil (null) it means it is a one time fee.
	// Example: "P1D12H"
	BillingCadence *datetime.ISODuration `json:"billingCadence"`
}

func (r *FlatFeeRateCard) Compatible(v RateCard) error {
	return RateCardWithOverlay{
		base:    r,
		overlay: v,
	}.Validate()
}

func (r *FlatFeeRateCard) GetBillingCadence() *datetime.ISODuration {
	return r.BillingCadence
}

func (r *FlatFeeRateCard) ChangeMeta(fn func(m RateCardMeta) (RateCardMeta, error)) error {
	var err error
	r.RateCardMeta, err = fn(r.RateCardMeta)
	if err != nil {
		return err
	}

	return r.Validate()
}

func (r *FlatFeeRateCard) Merge(v RateCard) error {
	if r.Type() != v.Type() {
		return errors.New("type mismatch")
	}

	vv, ok := v.(*FlatFeeRateCard)
	if !ok {
		return errors.New("failed to cast to FlatFeeRateCard")
	}

	r.RateCardMeta = vv.RateCardMeta
	r.BillingCadence = vv.BillingCadence

	return nil
}

func (r *FlatFeeRateCard) Type() RateCardType {
	return FlatFeeRateCardType
}

func (r *FlatFeeRateCard) Key() string {
	return r.RateCardMeta.Key
}

func (r *FlatFeeRateCard) Equal(v RateCard) bool {
	if r.Type() != v.Type() {
		return false
	}

	vv, ok := v.(*FlatFeeRateCard)
	if !ok {
		return false
	}

	if !r.RateCardMeta.Equal(vv.RateCardMeta) {
		return false
	}

	if lo.FromPtr(r.BillingCadence).ISOString() != lo.FromPtr(vv.BillingCadence).ISOString() {
		return false
	}

	return true
}

func (r *FlatFeeRateCard) AsMeta() RateCardMeta {
	return r.RateCardMeta
}

func (r *FlatFeeRateCard) ValidateWith(v ...models.ValidatorFunc[*FlatFeeRateCard]) error {
	return models.Validate(r, v...)
}

func (r *FlatFeeRateCard) Validate() error {
	var errs []error

	if err := r.RateCardMeta.Validate(); err != nil {
		errs = append(errs, err)
	}

	if r.BillingCadence != nil {
		if r.BillingCadence.IsNegative() || r.BillingCadence.IsZero() {
			errs = append(errs, ErrBillingCadenceInvalidValue)
		}

		// Billing Cadence has to be at least 1 hour
		if per, err := r.BillingCadence.Subtract(datetime.NewISODuration(0, 0, 0, 0, 1, 0, 0)); err == nil && per.Sign() == -1 {
			errs = append(errs, ErrBillingCadenceInvalidValue)
		}
	}

	if err := r.Discounts.Validate(); err != nil {
		errs = append(errs, err)
	}

	if r.Discounts.Usage != nil {
		errs = append(errs, ErrUsageDiscountWithFlatPrice)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (r *FlatFeeRateCard) Clone() RateCard {
	clone := &FlatFeeRateCard{
		RateCardMeta: r.RateCardMeta.Clone(),
	}

	if r.BillingCadence != nil {
		bc := *r.BillingCadence
		clone.BillingCadence = &bc
	}

	return clone
}

func (r *FlatFeeRateCard) MarshalJSON() ([]byte, error) {
	serde := struct {
		RateCardSerde
		RateCardMeta
		BillingCadence *datetime.ISODuration `json:"billingCadence"`
	}{
		RateCardMeta:   r.RateCardMeta,
		BillingCadence: r.BillingCadence,
		RateCardSerde: RateCardSerde{
			Type: r.Type(),
		},
	}

	return json.Marshal(serde)
}

var (
	_ RateCard                                    = (*UsageBasedRateCard)(nil)
	_ models.CustomValidator[*UsageBasedRateCard] = (*UsageBasedRateCard)(nil)
)

type UsageBasedRateCard struct {
	RateCardMeta

	// BillingCadence defines the billing cadence of the RateCard in ISO8601 format.
	// Example: "P1D12H"
	BillingCadence datetime.ISODuration `json:"billingCadence"`
}

func (r *UsageBasedRateCard) Compatible(v RateCard) error {
	return RateCardWithOverlay{
		base:    r,
		overlay: v,
	}.Validate()
}

func (r *UsageBasedRateCard) GetBillingCadence() *datetime.ISODuration {
	return &r.BillingCadence
}

func (r *UsageBasedRateCard) Clone() RateCard {
	clone := &UsageBasedRateCard{
		RateCardMeta:   r.RateCardMeta.Clone(),
		BillingCadence: r.BillingCadence,
	}

	return clone
}

func (r *UsageBasedRateCard) ChangeMeta(fn func(m RateCardMeta) (RateCardMeta, error)) error {
	var err error
	r.RateCardMeta, err = fn(r.RateCardMeta)
	if err != nil {
		return err
	}

	return r.Validate()
}

func (r *UsageBasedRateCard) Merge(v RateCard) error {
	if r.Type() != v.Type() {
		return errors.New("type mismatch")
	}

	vv, ok := v.(*UsageBasedRateCard)
	if !ok {
		return errors.New("failed to cast to UsageBasedRateCard")
	}

	r.RateCardMeta = vv.RateCardMeta
	r.BillingCadence = vv.BillingCadence

	return nil
}

func (r *UsageBasedRateCard) Type() RateCardType {
	return UsageBasedRateCardType
}

func (r *UsageBasedRateCard) Key() string {
	return r.RateCardMeta.Key
}

func (r *UsageBasedRateCard) Equal(v RateCard) bool {
	if r.Type() != v.Type() {
		return false
	}

	vv, ok := v.(*UsageBasedRateCard)
	if !ok {
		return false
	}

	if !r.RateCardMeta.Equal(vv.RateCardMeta) {
		return false
	}

	if r.BillingCadence.ISOString() != vv.BillingCadence.ISOString() {
		return false
	}

	return true
}

func (r *UsageBasedRateCard) AsMeta() RateCardMeta {
	return r.RateCardMeta
}

func (r *UsageBasedRateCard) ValidateWith(v ...models.ValidatorFunc[*UsageBasedRateCard]) error {
	return models.Validate(r, v...)
}

func (r *UsageBasedRateCard) Validate() error {
	var errs []error

	if err := r.RateCardMeta.Validate(); err != nil {
		errs = append(errs, err)
	}

	if r.BillingCadence.IsNegative() || r.BillingCadence.IsZero() {
		errs = append(errs, ErrBillingCadenceInvalidValue)
	}

	// Billing Cadence has to be at least 1 hour
	if per, err := r.BillingCadence.Subtract(datetime.NewISODuration(0, 0, 0, 0, 1, 0, 0)); err == nil && per.Sign() == -1 {
		errs = append(errs, ErrBillingCadenceInvalidValue)
	}

	if r.Price != nil && r.Price.Type() == FlatPriceType && r.Discounts.Usage != nil {
		errs = append(errs, ErrUsageDiscountWithFlatPrice)
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (r *UsageBasedRateCard) MarshalJSON() ([]byte, error) {
	serde := struct {
		RateCardSerde
		RateCardMeta
		BillingCadence datetime.ISODuration `json:"billingCadence"`
	}{
		RateCardMeta:   r.RateCardMeta,
		BillingCadence: r.BillingCadence,
		RateCardSerde: RateCardSerde{
			Type: r.Type(),
		},
	}

	return json.Marshal(serde)
}

var (
	_ models.Equaler[RateCards]         = (*RateCards)(nil)
	_ models.CustomValidator[RateCards] = (*RateCards)(nil)
)

type RateCards []RateCard

func (c RateCards) Clone() RateCards {
	clone := make(RateCards, len(c))
	for i, rc := range c {
		clone[i] = rc.Clone()
	}
	return clone
}

func (c RateCards) At(idx int) RateCard {
	return c[idx]
}

func (c RateCards) Billables() RateCards {
	var billables RateCards
	for _, rc := range c {
		// An effective price of 0 is still counted as a billable item
		if rc.AsMeta().Price != nil {
			billables = append(billables, rc)
		}
	}

	return billables
}

// SingleBillingCadence returns true if all ratecards in the collection has the same billing cadence.
func (c RateCards) SingleBillingCadence() bool {
	m := make(map[datetime.ISODurationString]struct{})

	for _, rc := range c {
		// An effective price of 0 is still counted as a billable item
		if rc.AsMeta().Price != nil {
			// One time prices are excluded
			if bc := rc.GetBillingCadence(); bc != nil {
				m[bc.Normalise(true).ISOString()] = struct{}{}
			}
		}
	}

	return len(m) <= 1
}

func (c RateCards) Equal(v RateCards) bool {
	if len(c) != len(v) {
		return false
	}

	leftSet := make(map[string]RateCard)
	for _, rc := range c {
		leftSet[rc.Key()] = rc
	}

	rightSet := make(map[string]RateCard)
	for _, rc := range v {
		rightSet[rc.Key()] = rc
	}

	if len(leftSet) != len(rightSet) {
		return false
	}

	var visited int
	for key, left := range leftSet {
		right, ok := rightSet[key]
		if !ok {
			return false
		}

		if !left.Equal(right) {
			return false
		}

		visited++
	}

	return visited == len(rightSet)
}

func (c RateCards) ValidateWith(v ...models.ValidatorFunc[RateCards]) error {
	return models.Validate(c, v...)
}

// ValidateRateCards returns a validation function can be passed to the object
// which implements models.CustomValidator interface.
// It checks for invalid and duplicated ratecards in the RateCards collection.
func ValidateRateCards() models.ValidatorFunc[RateCards] {
	return func(ratecards RateCards) error {
		var errs []error

		rateCardKeys := make(map[string]RateCard)

		for _, rateCard := range ratecards {
			fieldSelector := models.NewFieldSelectorGroup(
				models.NewFieldSelector("ratecards").WithExpression(
					models.NewFieldAttrValue("key", rateCard.Key())),
			)

			if _, ok := rateCardKeys[rateCard.Key()]; ok {
				errs = append(errs, models.ErrorWithFieldPrefix(fieldSelector, ErrRateCardDuplicatedKey))
			} else {
				rateCardKeys[rateCard.Key()] = rateCard
			}

			if err := rateCard.Validate(); err != nil {
				errs = append(errs, models.ErrorWithFieldPrefix(fieldSelector, err))
			}
		}

		return errors.Join(errs...)
	}
}

func (c RateCards) Validate() error {
	return c.ValidateWith(ValidateRateCards())
}

type RateCardWithOverlay struct {
	base    RateCard
	overlay RateCard
}

func NewRateCardWithOverlay(base, overlay RateCard) RateCardWithOverlay {
	return RateCardWithOverlay{
		base:    base,
		overlay: overlay,
	}
}

func (r RateCardWithOverlay) ValidateWith(validators ...models.ValidatorFunc[RateCardWithOverlay]) error {
	return models.Validate(r, validators...)
}

func (r RateCardWithOverlay) Validate() error {
	if r.base == nil || r.overlay == nil {
		return nil
	}

	return r.ValidateWith(
		ValidateRateCardsShareSameKey,
		ValidateRateCardsHaveCompatiblePrice,
		ValidateRateCardsHaveCompatibleFeatureKey,
		ValidateRateCardsHaveCompatibleFeatureID,
		ValidateRateCardsHaveCompatibleBillingCadence,
		ValidateRateCardsHaveCompatibleEntitlementTemplate,
		ValidateRateCardsHaveCompatibleDiscounts,
	)
}

var ValidateRateCardsShareSameKey = models.ValidatorFunc[RateCardWithOverlay](func(r RateCardWithOverlay) error {
	if r.base == nil || r.overlay == nil {
		return nil
	}

	fieldSelector := models.NewFieldSelectorGroup(models.NewFieldSelector("ratecards").
		WithExpression(models.NewFieldAttrValue("key", r.base.Key())))

	if r.base.Key() != r.overlay.Key() {
		return models.ErrorWithFieldPrefix(fieldSelector, ErrRateCardKeyMismatch)
	}

	return nil
})

var ValidateRateCardsHaveCompatiblePrice = models.ValidatorFunc[RateCardWithOverlay](func(r RateCardWithOverlay) error {
	if r.base == nil || r.overlay == nil {
		return nil
	}

	var errs []error

	rMeta, vMeta := r.base.AsMeta(), r.overlay.AsMeta()

	fieldSelector := models.NewFieldSelectorGroup(models.NewFieldSelector("ratecards").
		WithExpression(models.NewFieldAttrValue("key", r.base.Key())))

	// Validate Price
	if rMeta.Price != nil && vMeta.Price != nil {
		if rMeta.Price.Type() != vMeta.Price.Type() {
			errs = append(errs, models.ErrorWithFieldPrefix(fieldSelector, ErrRateCardPriceTypeMismatch))
		}

		switch rMeta.Price.Type() {
		case FlatPriceType:
			rFlat, _ := rMeta.Price.AsFlat()
			vFlat, _ := vMeta.Price.AsFlat()

			if rFlat.PaymentTerm != vFlat.PaymentTerm {
				errs = append(errs, models.ErrorWithFieldPrefix(fieldSelector, ErrRateCardPricePaymentTermMismatch))
			}
		default:
			errs = append(errs, models.ErrorWithFieldPrefix(fieldSelector, ErrRateCardOnlyFlatPriceAllowed))
		}
	}

	return errors.Join(errs...)
})

var ValidateRateCardsHaveCompatibleFeatureKey = models.ValidatorFunc[RateCardWithOverlay](func(r RateCardWithOverlay) error {
	if r.base == nil || r.overlay == nil {
		return nil
	}

	rMeta, vMeta := r.base.AsMeta(), r.overlay.AsMeta()

	fieldSelector := models.NewFieldSelectorGroup(models.NewFieldSelector("ratecards").
		WithExpression(models.NewFieldAttrValue("key", r.base.Key())))

	if rMeta.FeatureKey != nil && vMeta.FeatureKey != nil && *rMeta.FeatureKey != *vMeta.FeatureKey {
		return models.ErrorWithFieldPrefix(fieldSelector, ErrRateCardFeatureKeyMismatch)
	}

	return nil
})

var ValidateRateCardsHaveCompatibleFeatureID = models.ValidatorFunc[RateCardWithOverlay](func(r RateCardWithOverlay) error {
	if r.base == nil || r.overlay == nil {
		return nil
	}

	rMeta, vMeta := r.base.AsMeta(), r.overlay.AsMeta()

	fieldSelector := models.NewFieldSelectorGroup(models.NewFieldSelector("ratecards").
		WithExpression(models.NewFieldAttrValue("key", r.base.Key())))

	if rMeta.FeatureID != nil && vMeta.FeatureID != nil && *rMeta.FeatureID != *vMeta.FeatureID {
		return models.ErrorWithFieldPrefix(fieldSelector, ErrRateCardFeatureIDMismatch)
	}

	return nil
})

var ValidateRateCardsHaveCompatibleBillingCadence = models.ValidatorFunc[RateCardWithOverlay](func(r RateCardWithOverlay) error {
	if r.base == nil || r.overlay == nil {
		return nil
	}

	var errs []error

	rBillingCadence, vBillingCadence := r.base.GetBillingCadence(), r.overlay.GetBillingCadence()

	fieldSelector := models.NewFieldSelectorGroup(models.NewFieldSelector("ratecards").
		WithExpression(models.NewFieldAttrValue("key", r.base.Key())))

	if rBillingCadence != nil && vBillingCadence != nil && !rBillingCadence.Equal(vBillingCadence) {
		errs = append(errs, models.ErrorWithFieldPrefix(fieldSelector, ErrRateCardBillingCadenceMismatch))
	}

	return errors.Join(errs...)
})

var ValidateRateCardsHaveCompatibleEntitlementTemplate = models.ValidatorFunc[RateCardWithOverlay](func(r RateCardWithOverlay) error {
	if r.base == nil || r.overlay == nil {
		return nil
	}

	var errs []error

	rMeta, vMeta := r.base.AsMeta(), r.overlay.AsMeta()

	if rMeta.EntitlementTemplate != nil && vMeta.EntitlementTemplate != nil {
		if rMeta.EntitlementTemplate.Type() != vMeta.EntitlementTemplate.Type() {
			errs = append(errs, ErrRateCardEntitlementTemplateTypeMismatch)
		} else {
			switch rMeta.EntitlementTemplate.Type() {
			case entitlement.EntitlementTypeStatic:
				errs = append(errs, ErrRateCardStaticEntitlementTemplateNotAllowed)
			case entitlement.EntitlementTypeMetered:
				rMetered, err := rMeta.EntitlementTemplate.AsMetered()
				if err != nil {
					return err
				}

				vMetered, err := vMeta.EntitlementTemplate.AsMetered()
				if err != nil {
					return err
				}

				if !rMetered.UsagePeriod.Equal(&vMetered.UsagePeriod) {
					errs = append(errs, ErrRateCardMeteredEntitlementTemplateUsagePeriodMismatch)
				}
			case entitlement.EntitlementTypeBoolean:
			}
		}
	}

	err := errors.Join(errs...)
	if err != nil {
		fieldSelector := models.NewFieldSelectorGroup(
			models.NewFieldSelector("ratecards").
				WithExpression(models.NewFieldAttrValue("key", r.base.Key())),
			models.NewFieldSelector("entitlementTemplate"),
		)

		return models.ErrorWithFieldPrefix(fieldSelector, err)
	}

	return nil
})

var ValidateRateCardsHaveCompatibleDiscounts = models.ValidatorFunc[RateCardWithOverlay](func(r RateCardWithOverlay) error {
	if r.base == nil || r.overlay == nil {
		return nil
	}

	var errs []error

	rMeta, vMeta := r.base.AsMeta(), r.overlay.AsMeta()

	if rMeta.Discounts.Percentage != nil && vMeta.Discounts.Percentage != nil {
		fieldSelector := models.NewFieldSelectorGroup(models.NewFieldSelector("discounts"))

		errs = append(errs, models.ErrorWithFieldPrefix(fieldSelector, ErrRateCardPercentageDiscountNotAllowed))
	}

	err := errors.Join(errs...)
	if err != nil {
		fieldSelector := models.NewFieldSelectorGroup(
			models.NewFieldSelector("ratecards").
				WithExpression(models.NewFieldAttrValue("key", r.base.Key())),
		)

		return models.ErrorWithFieldPrefix(fieldSelector, err)
	}

	return nil
})

func ValidateRateCardsWithFeatures(ctx context.Context, resolver NamespacedFeatureResolver) func(cards RateCards) error {
	return func(rateCards RateCards) error {
		var errs []error

		for _, rateCard := range rateCards {
			rc := rateCard.AsMeta()

			rateCardFieldSelector := models.NewFieldSelectorGroup(
				models.NewFieldSelector("rateCards").
					WithExpression(
						models.NewFieldAttrValue("key", rateCard.Key()),
					),
			)

			if rc.FeatureID == nil && rc.FeatureKey == nil {
				continue
			}

			feat, err := resolver.Resolve(ctx, rc.FeatureID, rc.FeatureKey)
			if err != nil {
				switch {
				case models.IsGenericNotFoundError(err):
					errs = append(errs, models.ErrorWithFieldPrefix(rateCardFieldSelector, ErrRateCardFeatureNotFound))
				case models.IsGenericConflictError(err):
					errs = append(errs, models.ErrorWithFieldPrefix(rateCardFieldSelector, ErrRateCardFeatureMismatch))
				default:
					errs = append(errs, fmt.Errorf("failed to resolve feature for ratecard: %w", err))
				}

				continue
			}

			if feat.ArchivedAt != nil && clock.Now().UTC().After(feat.ArchivedAt.UTC()) {
				errs = append(errs, models.ErrorWithFieldPrefix(rateCardFieldSelector, ErrRateCardFeatureArchived))
			}
		}

		return errors.Join(errs...)
	}
}
