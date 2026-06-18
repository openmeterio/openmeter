package intentoverride

import (
	"errors"
	"fmt"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type FlatFeeMixin = entutils.RecursiveMixin[flatFeeMixin]

type flatFeeMixin struct {
	mixin.Schema
}

func (flatFeeMixin) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

func (flatFeeMixin) Fields() []ent.Field {
	return []ent.Field{
		field.String("override_feature_key").
			Optional().
			Nillable(),
		field.Enum("override_payment_term").
			GoType(productcatalog.PaymentTermType("")).
			Optional().
			Nillable(),
		field.String("override_pro_rating").
			GoType(&productcatalog.ProRatingConfig{}).
			ValueScanner(entutils.JSONStringValueScanner[*productcatalog.ProRatingConfig]()).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Nillable(),
		field.Other("override_amount_before_proration", alpacadecimal.Decimal{}).
			SchemaType(map[string]string{
				dialect.Postgres: "numeric",
			}).
			Optional().
			Nillable(),
		field.String("override_percentage_discounts").
			GoType(&PercentageDiscountsOverride{}).
			ValueScanner(entutils.JSONStringValueScanner[*PercentageDiscountsOverride]()).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Nillable(),
	}
}

type PercentageDiscountsOverride struct {
	Value *productcatalog.PercentageDiscount `json:"value,omitempty"`
}

func (p *PercentageDiscountsOverride) Validate() error {
	if p == nil || p.Value == nil {
		return nil
	}

	return p.Value.Validate()
}

type FlatFee struct {
	OverrideBase

	// FeatureKey has three states: None means not overridden, Some(nil) means cleared,
	// and Some(value) means overridden to value.
	FeatureKey            mo.Option[*string]              `json:"featureKey,omitzero"`
	PaymentTerm           *productcatalog.PaymentTermType `json:"paymentTerm,omitempty"`
	ProRating             *productcatalog.ProRatingConfig `json:"proRating,omitempty"`
	AmountBeforeProration *alpacadecimal.Decimal          `json:"amountBeforeProration,omitempty"`
	// PercentageDiscounts has three states: None means not overridden, Some(nil) means cleared,
	// and Some(value) means overridden to value.
	PercentageDiscounts mo.Option[*productcatalog.PercentageDiscount] `json:"percentageDiscounts,omitzero"`
}

func (o FlatFee) Normalized() FlatFee {
	o.OverrideBase = o.OverrideBase.Normalized()

	return o
}

func (o FlatFee) Validate() error {
	var errs []error

	if err := o.OverrideBase.Validate(); err != nil {
		errs = append(errs, err)
	}

	if o.FeatureKey.IsPresent() {
		featureKey := o.FeatureKey.OrEmpty()
		if featureKey != nil && *featureKey == "" {
			errs = append(errs, errors.New("feature key cannot be empty"))
		}
	}

	if o.PaymentTerm != nil {
		if err := o.PaymentTerm.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("payment term: %w", err))
		}
	}

	if o.ProRating != nil {
		if err := o.ProRating.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("pro rating: %w", err))
		}
	}

	if o.AmountBeforeProration != nil && o.AmountBeforeProration.IsNegative() {
		errs = append(errs, errors.New("amount before proration cannot be negative"))
	}

	if o.PercentageDiscounts.IsPresent() {
		percentageDiscounts := o.PercentageDiscounts.OrEmpty()
		if percentageDiscounts != nil {
			if err := percentageDiscounts.Validate(); err != nil {
				errs = append(errs, fmt.Errorf("percentage discounts: %w", err))
			}
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type FlatFeeCreator[T any] interface {
	BaseCreator[T]

	SetOverrideFeatureKey(featureKey string) T
	SetOverridePaymentTerm(paymentTerm productcatalog.PaymentTermType) T
	SetOverrideProRating(proRating *productcatalog.ProRatingConfig) T
	SetOverrideAmountBeforeProration(amountBeforeProration alpacadecimal.Decimal) T
	SetOverridePercentageDiscounts(percentageDiscounts *PercentageDiscountsOverride) T
}

func CreateFlatFee[T FlatFeeCreator[T]](creator T, override *FlatFee) (T, error) {
	if override == nil {
		return creator, nil
	}

	normalized := override.Normalized()
	if err := normalized.Validate(); err != nil {
		var empty T
		return empty, err
	}

	creator, err := CreateBase(creator, &normalized.OverrideBase)
	if err != nil {
		var empty T
		return empty, err
	}

	if normalized.FeatureKey.IsPresent() {
		featureKey := normalized.FeatureKey.OrEmpty()
		if featureKey != nil {
			creator = creator.SetOverrideFeatureKey(*featureKey)
		} else {
			creator = creator.SetOverrideFeatureKey("")
		}
	}

	if normalized.PaymentTerm != nil {
		creator = creator.SetOverridePaymentTerm(*normalized.PaymentTerm)
	}

	if normalized.ProRating != nil {
		creator = creator.SetOverrideProRating(normalized.ProRating)
	}

	if normalized.AmountBeforeProration != nil {
		creator = creator.SetOverrideAmountBeforeProration(*normalized.AmountBeforeProration)
	}

	if normalized.PercentageDiscounts.IsPresent() {
		creator = creator.SetOverridePercentageDiscounts(&PercentageDiscountsOverride{
			Value: normalized.PercentageDiscounts.OrEmpty(),
		})
	}

	return creator, nil
}

type FlatFeeUpdater[T any] interface {
	BaseUpdater[T]

	SetOrClearOverrideFeatureKey(featureKey *string) T
	ClearOverrideFeatureKey() T
	SetOrClearOverridePaymentTerm(paymentTerm *productcatalog.PaymentTermType) T
	ClearOverridePaymentTerm() T
	SetOrClearOverrideProRating(proRating **productcatalog.ProRatingConfig) T
	ClearOverrideProRating() T
	SetOrClearOverrideAmountBeforeProration(amountBeforeProration *alpacadecimal.Decimal) T
	ClearOverrideAmountBeforeProration() T
	SetOverridePercentageDiscounts(percentageDiscounts *PercentageDiscountsOverride) T
	ClearOverridePercentageDiscounts() T
}

func UpdateFlatFee[T FlatFeeUpdater[T]](updater T, override *FlatFee) (T, error) {
	if override == nil {
		updater = clearOnBaseUpdater(updater)
		return updater.
			ClearOverrideFeatureKey().
			ClearOverridePaymentTerm().
			ClearOverrideProRating().
			ClearOverrideAmountBeforeProration().
			ClearOverridePercentageDiscounts(), nil
	}

	normalized := override.Normalized()
	if err := normalized.Validate(); err != nil {
		var empty T
		return empty, err
	}

	updater, err := UpdateBase(updater, &normalized.OverrideBase)
	if err != nil {
		var empty T
		return empty, err
	}

	updater = updater.
		SetOrClearOverrideFeatureKey(stringPtrToDB(normalized.FeatureKey)).
		SetOrClearOverridePaymentTerm(normalized.PaymentTerm).
		SetOrClearOverrideProRating(fromOptionalPtrToSetOrClear(normalized.ProRating)).
		SetOrClearOverrideAmountBeforeProration(normalized.AmountBeforeProration)

	if normalized.PercentageDiscounts.IsPresent() {
		updater = updater.SetOverridePercentageDiscounts(&PercentageDiscountsOverride{
			Value: normalized.PercentageDiscounts.OrEmpty(),
		})
	} else {
		updater = updater.ClearOverridePercentageDiscounts()
	}

	return updater, nil
}

type FlatFeeGetter[T any] interface {
	BaseGetter[T]

	GetOverrideFeatureKey() *string
	GetOverridePaymentTerm() *productcatalog.PaymentTermType
	GetOverrideProRating() *productcatalog.ProRatingConfig
	GetOverrideAmountBeforeProration() *alpacadecimal.Decimal
	GetOverridePercentageDiscounts() *PercentageDiscountsOverride
}

func MapFlatFeeFromDB[T FlatFeeGetter[T]](entity T) *FlatFee {
	base := MapBaseFromDB(entity)
	if base == nil {
		return nil
	}

	percentageDiscounts := mo.None[*productcatalog.PercentageDiscount]()
	if overridePercentageDiscounts := entity.GetOverridePercentageDiscounts(); overridePercentageDiscounts != nil {
		percentageDiscounts = mo.Some(overridePercentageDiscounts.Value)
	}

	override := FlatFee{
		OverrideBase:          *base,
		FeatureKey:            optionStringPtrFromDB(entity.GetOverrideFeatureKey()),
		PaymentTerm:           entity.GetOverridePaymentTerm(),
		ProRating:             entity.GetOverrideProRating(),
		AmountBeforeProration: entity.GetOverrideAmountBeforeProration(),
		PercentageDiscounts:   percentageDiscounts,
	}

	return &override
}
