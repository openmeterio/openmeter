package intentoverride

import (
	"errors"
	"fmt"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type UsageBasedMixin = entutils.RecursiveMixin[usageBasedMixin]

type usageBasedMixin struct {
	mixin.Schema
}

func (usageBasedMixin) Mixin() []ent.Mixin {
	return []ent.Mixin{
		BaseMixin{},
	}
}

func (usageBasedMixin) Fields() []ent.Field {
	return []ent.Field{
		field.String("override_feature_key").
			Optional().
			NotEmpty().
			Nillable(),
		field.String("override_price").
			GoType(&productcatalog.Price{}).
			ValueScanner(entutils.JSONStringValueScanner[*productcatalog.Price]()).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Nillable(),
		field.String("override_discounts").
			GoType(&productcatalog.Discounts{}).
			ValueScanner(entutils.JSONStringValueScanner[*productcatalog.Discounts]()).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Nillable(),
	}
}

type UsageBased struct {
	OverrideBase

	FeatureKey *string                   `json:"featureKey,omitempty"`
	Price      *productcatalog.Price     `json:"price,omitempty"`
	Discounts  *productcatalog.Discounts `json:"discounts,omitempty"`
}

func (o UsageBased) Normalized() UsageBased {
	o.OverrideBase = o.OverrideBase.Normalized()

	return o
}

func (o UsageBased) Validate() error {
	var errs []error

	if err := o.OverrideBase.Validate(); err != nil {
		errs = append(errs, err)
	}

	if o.FeatureKey != nil && *o.FeatureKey == "" {
		errs = append(errs, errors.New("feature key cannot be empty"))
	}

	if o.Price != nil {
		if err := o.Price.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("price: %w", err))
		}
	}

	if o.Discounts != nil {
		if err := o.Discounts.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("discounts: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type UsageBasedCreator[T any] interface {
	BaseCreator[T]

	SetOverrideFeatureKey(featureKey string) T
	SetOverridePrice(price *productcatalog.Price) T
	SetOverrideDiscounts(discounts *productcatalog.Discounts) T
}

func CreateUsageBased[T UsageBasedCreator[T]](creator T, override *UsageBased) (T, error) {
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

	if normalized.FeatureKey != nil {
		creator = creator.SetOverrideFeatureKey(*normalized.FeatureKey)
	}

	if normalized.Price != nil {
		creator = creator.SetOverridePrice(normalized.Price)
	}

	if normalized.Discounts != nil {
		creator = creator.SetOverrideDiscounts(normalized.Discounts)
	}

	return creator, nil
}

type UsageBasedUpdater[T any] interface {
	BaseUpdater[T]

	SetOrClearOverrideFeatureKey(featureKey *string) T
	ClearOverrideFeatureKey() T
	SetOrClearOverridePrice(price **productcatalog.Price) T
	ClearOverridePrice() T
	SetOrClearOverrideDiscounts(discounts **productcatalog.Discounts) T
	ClearOverrideDiscounts() T
}

func UpdateUsageBased[T UsageBasedUpdater[T]](updater T, override *UsageBased) (T, error) {
	if override == nil {
		updater = clearOnBaseUpdater(updater)
		return updater.
			ClearOverrideFeatureKey().
			ClearOverridePrice().
			ClearOverrideDiscounts(), nil
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

	return updater.
		SetOrClearOverrideFeatureKey(normalized.FeatureKey).
		SetOrClearOverridePrice(fromOptionalPtrToSetOrClear(normalized.Price)).
		SetOrClearOverrideDiscounts(fromOptionalPtrToSetOrClear(normalized.Discounts)), nil
}

type UsageBasedGetter[T any] interface {
	BaseGetter[T]

	GetOverrideFeatureKey() *string
	GetOverridePrice() *productcatalog.Price
	GetOverrideDiscounts() *productcatalog.Discounts
}

func MapUsageBasedFromDB[T UsageBasedGetter[T]](entity T) *UsageBased {
	base := MapBaseFromDB(entity)
	if base == nil {
		return nil
	}

	override := UsageBased{
		OverrideBase: *base,
		FeatureKey:   entity.GetOverrideFeatureKey(),
		Price:        entity.GetOverridePrice(),
		Discounts:    entity.GetOverrideDiscounts(),
	}

	return &override
}
