package intentoverride

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type BaseMixin = entutils.RecursiveMixin[overrideBaseMixin]

type overrideBaseMixin struct {
	mixin.Schema
}

func (overrideBaseMixin) Mixin() []ent.Mixin {
	return nil
}

type Kind string

const (
	KindEdit   Kind = "edit"
	KindDelete Kind = "delete"
)

func (k Kind) Values() []string {
	return []string{
		string(KindEdit),
		string(KindDelete),
	}
}

func (k Kind) Validate() error {
	if !slices.Contains(k.Values(), string(k)) {
		return models.NewGenericValidationError(fmt.Errorf("invalid intent override kind: %s", k))
	}

	return nil
}

type TaxBehaviorOverride string

const (
	TaxBehaviorOverrideClear TaxBehaviorOverride = ""
)

func (t TaxBehaviorOverride) Validate() error {
	if t == TaxBehaviorOverrideClear {
		return nil
	}

	taxBehavior := productcatalog.TaxBehavior(t)
	if err := taxBehavior.Validate(); err != nil {
		return models.NewGenericValidationError(fmt.Errorf("invalid tax behavior override: %s", t))
	}

	return nil
}

type OverrideBase struct {
	Kind Kind `json:"kind"`

	Name *string `json:"name,omitempty"`
	// Description has three states: None means not overridden, Some(nil) means cleared,
	// and Some(value) means overridden to value.
	Description mo.Option[*string] `json:"description,omitzero"`
	Metadata    *models.Metadata   `json:"metadata,omitempty"`
	// TaxBehavior has three states: None means not overridden, Some(nil) means cleared,
	// and Some(value) means overridden to value.
	TaxBehavior mo.Option[*productcatalog.TaxBehavior] `json:"taxBehavior,omitzero"`
	TaxCodeID   *string                                `json:"taxCodeID,omitempty"`

	ServicePeriod     *timeutil.ClosedPeriod `json:"servicePeriod,omitempty"`
	FullServicePeriod *timeutil.ClosedPeriod `json:"fullServicePeriod,omitempty"`
	BillingPeriod     *timeutil.ClosedPeriod `json:"billingPeriod,omitempty"`
}

func (o OverrideBase) Normalized() OverrideBase {
	if o.ServicePeriod != nil {
		servicePeriod := meta.NormalizeClosedPeriod(*o.ServicePeriod)
		o.ServicePeriod = &servicePeriod
	}

	if o.FullServicePeriod != nil {
		fullServicePeriod := meta.NormalizeClosedPeriod(*o.FullServicePeriod)
		o.FullServicePeriod = &fullServicePeriod
	}

	if o.BillingPeriod != nil {
		billingPeriod := meta.NormalizeClosedPeriod(*o.BillingPeriod)
		o.BillingPeriod = &billingPeriod
	}

	return o
}

func (o OverrideBase) Validate() error {
	var errs []error

	if err := o.Kind.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("kind: %w", err))
	}

	if o.Name != nil && *o.Name == "" {
		errs = append(errs, errors.New("name cannot be empty"))
	}

	if o.TaxBehavior.IsPresent() {
		taxBehavior := o.TaxBehavior.OrEmpty()
		if taxBehavior != nil {
			if err := taxBehavior.Validate(); err != nil {
				errs = append(errs, fmt.Errorf("tax behavior: %w", err))
			}
		}
	}

	if o.ServicePeriod != nil {
		if err := o.ServicePeriod.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("service period: %w", err))
		}
	}

	if o.FullServicePeriod != nil {
		if err := o.FullServicePeriod.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("full service period: %w", err))
		}
	}

	if o.BillingPeriod != nil {
		if err := o.BillingPeriod.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("billing period: %w", err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (overrideBaseMixin) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("override_kind").
			GoType(Kind("")).
			Optional().
			Nillable(),
		field.String("override_name").
			Optional().
			NotEmpty().
			Nillable(),
		field.String("override_description").
			Optional().
			Nillable(),
		field.String("override_metadata").
			GoType(&models.Metadata{}).
			ValueScanner(entutils.JSONStringValueScanner[*models.Metadata]()).
			SchemaType(map[string]string{
				dialect.Postgres: "jsonb",
			}).
			Optional().
			Nillable(),
		field.String("override_tax_behavior").
			GoType(TaxBehaviorOverride("")).
			Validate(func(taxBehavior string) error {
				return TaxBehaviorOverride(taxBehavior).Validate()
			}).
			Optional().
			Nillable(),
		field.String("override_tax_code_id").
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}).
			Optional().
			Nillable(),
		field.Time("override_service_period_from").
			Optional().
			Nillable(),
		field.Time("override_service_period_to").
			Optional().
			Nillable(),
		field.Time("override_full_service_period_from").
			Optional().
			Nillable(),
		field.Time("override_full_service_period_to").
			Optional().
			Nillable(),
		field.Time("override_billing_period_from").
			Optional().
			Nillable(),
		field.Time("override_billing_period_to").
			Optional().
			Nillable(),
	}
}

type BaseCreator[T any] interface {
	SetOverrideKind(kind Kind) T
	SetOverrideName(name string) T
	SetOverrideDescription(description string) T
	SetOverrideMetadata(metadata *models.Metadata) T
	SetOverrideTaxBehavior(taxBehavior TaxBehaviorOverride) T
	SetOverrideTaxCodeID(taxCodeID string) T
	SetOverrideServicePeriodFrom(servicePeriodFrom time.Time) T
	SetOverrideServicePeriodTo(servicePeriodTo time.Time) T
	SetOverrideFullServicePeriodFrom(fullServicePeriodFrom time.Time) T
	SetOverrideFullServicePeriodTo(fullServicePeriodTo time.Time) T
	SetOverrideBillingPeriodFrom(billingPeriodFrom time.Time) T
	SetOverrideBillingPeriodTo(billingPeriodTo time.Time) T
}

func CreateBase[T BaseCreator[T]](creator T, override *OverrideBase) (T, error) {
	if override == nil {
		return creator, nil
	}

	normalized := override.Normalized()
	if err := normalized.Validate(); err != nil {
		var empty T
		return empty, err
	}

	creator = creator.SetOverrideKind(normalized.Kind)

	if normalized.Name != nil {
		creator = creator.SetOverrideName(*normalized.Name)
	}

	if normalized.Description.IsPresent() {
		description := normalized.Description.OrEmpty()
		if description != nil {
			creator = creator.SetOverrideDescription(*description)
		} else {
			creator = creator.SetOverrideDescription("")
		}
	}

	if normalized.Metadata != nil {
		creator = creator.SetOverrideMetadata(normalized.Metadata)
	}

	if normalized.TaxBehavior.IsPresent() {
		creator = creator.SetOverrideTaxBehavior(taxBehaviorOverrideFromOption(normalized.TaxBehavior))
	}

	if normalized.TaxCodeID != nil {
		creator = creator.SetOverrideTaxCodeID(*normalized.TaxCodeID)
	}

	if normalized.ServicePeriod != nil {
		creator = creator.
			SetOverrideServicePeriodFrom(normalized.ServicePeriod.From.UTC()).
			SetOverrideServicePeriodTo(normalized.ServicePeriod.To.UTC())
	}

	if normalized.FullServicePeriod != nil {
		creator = creator.
			SetOverrideFullServicePeriodFrom(normalized.FullServicePeriod.From.UTC()).
			SetOverrideFullServicePeriodTo(normalized.FullServicePeriod.To.UTC())
	}

	if normalized.BillingPeriod != nil {
		creator = creator.
			SetOverrideBillingPeriodFrom(normalized.BillingPeriod.From.UTC()).
			SetOverrideBillingPeriodTo(normalized.BillingPeriod.To.UTC())
	}

	return creator, nil
}

type BaseUpdater[T any] interface {
	SetOverrideKind(kind Kind) T
	ClearOverrideKind() T
	SetOrClearOverrideName(name *string) T
	ClearOverrideName() T
	SetOrClearOverrideDescription(description *string) T
	ClearOverrideDescription() T
	SetOrClearOverrideMetadata(metadata **models.Metadata) T
	ClearOverrideMetadata() T
	SetOverrideTaxBehavior(taxBehavior TaxBehaviorOverride) T
	ClearOverrideTaxBehavior() T
	SetOrClearOverrideTaxCodeID(taxCodeID *string) T
	ClearOverrideTaxCodeID() T
	SetOverrideServicePeriodFrom(servicePeriodFrom time.Time) T
	ClearOverrideServicePeriodFrom() T
	SetOverrideServicePeriodTo(servicePeriodTo time.Time) T
	ClearOverrideServicePeriodTo() T
	SetOverrideFullServicePeriodFrom(fullServicePeriodFrom time.Time) T
	ClearOverrideFullServicePeriodFrom() T
	SetOverrideFullServicePeriodTo(fullServicePeriodTo time.Time) T
	ClearOverrideFullServicePeriodTo() T
	SetOverrideBillingPeriodFrom(billingPeriodFrom time.Time) T
	ClearOverrideBillingPeriodFrom() T
	SetOverrideBillingPeriodTo(billingPeriodTo time.Time) T
	ClearOverrideBillingPeriodTo() T
}

func UpdateBase[T BaseUpdater[T]](updater T, override *OverrideBase) (T, error) {
	if override == nil {
		return clearOnBaseUpdater(updater), nil
	}

	normalized := override.Normalized()
	if err := normalized.Validate(); err != nil {
		var empty T
		return empty, err
	}

	updater = updater.SetOverrideKind(normalized.Kind).
		SetOrClearOverrideName(normalized.Name).
		SetOrClearOverrideMetadata(fromOptionalPtrToSetOrClear(normalized.Metadata)).
		SetOrClearOverrideTaxCodeID(normalized.TaxCodeID).
		// Note: empty means clear, null means not overridden
		SetOrClearOverrideDescription(stringPtrToDB(normalized.Description))

	if normalized.TaxBehavior.IsPresent() {
		taxBehavior := taxBehaviorOverrideFromOption(normalized.TaxBehavior)
		updater = updater.SetOverrideTaxBehavior(taxBehavior)
	} else {
		updater = updater.ClearOverrideTaxBehavior()
	}

	if normalized.ServicePeriod != nil {
		updater = updater.
			SetOverrideServicePeriodFrom(normalized.ServicePeriod.From.UTC()).
			SetOverrideServicePeriodTo(normalized.ServicePeriod.To.UTC())
	} else {
		updater = updater.
			ClearOverrideServicePeriodFrom().
			ClearOverrideServicePeriodTo()
	}

	if normalized.FullServicePeriod != nil {
		updater = updater.
			SetOverrideFullServicePeriodFrom(normalized.FullServicePeriod.From.UTC()).
			SetOverrideFullServicePeriodTo(normalized.FullServicePeriod.To.UTC())
	} else {
		updater = updater.
			ClearOverrideFullServicePeriodFrom().
			ClearOverrideFullServicePeriodTo()
	}

	if normalized.BillingPeriod != nil {
		updater = updater.
			SetOverrideBillingPeriodFrom(normalized.BillingPeriod.From.UTC()).
			SetOverrideBillingPeriodTo(normalized.BillingPeriod.To.UTC())
	} else {
		updater = updater.
			ClearOverrideBillingPeriodFrom().
			ClearOverrideBillingPeriodTo()
	}

	return updater, nil
}

type BaseGetter[T any] interface {
	GetOverrideKind() *Kind
	GetOverrideName() *string
	GetOverrideDescription() *string
	GetOverrideMetadata() *models.Metadata
	GetOverrideTaxBehavior() *TaxBehaviorOverride
	GetOverrideTaxCodeID() *string
	GetOverrideServicePeriodFrom() *time.Time
	GetOverrideServicePeriodTo() *time.Time
	GetOverrideFullServicePeriodFrom() *time.Time
	GetOverrideFullServicePeriodTo() *time.Time
	GetOverrideBillingPeriodFrom() *time.Time
	GetOverrideBillingPeriodTo() *time.Time
}

func MapBaseFromDB[T BaseGetter[T]](entity T) *OverrideBase {
	if entity.GetOverrideKind() == nil {
		return nil
	}

	taxBehavior := mo.None[*productcatalog.TaxBehavior]()
	if overrideTaxBehavior := entity.GetOverrideTaxBehavior(); overrideTaxBehavior != nil {
		if *overrideTaxBehavior == TaxBehaviorOverrideClear {
			taxBehavior = mo.Some((*productcatalog.TaxBehavior)(nil))
		} else {
			value := productcatalog.TaxBehavior(*overrideTaxBehavior)
			taxBehavior = mo.Some(&value)
		}
	}

	override := OverrideBase{
		Kind:              *entity.GetOverrideKind(),
		Name:              entity.GetOverrideName(),
		Description:       optionStringPtrFromDB(entity.GetOverrideDescription()),
		Metadata:          entity.GetOverrideMetadata(),
		TaxBehavior:       taxBehavior,
		TaxCodeID:         entity.GetOverrideTaxCodeID(),
		ServicePeriod:     closedPeriodOptionFromDB(entity.GetOverrideServicePeriodFrom(), entity.GetOverrideServicePeriodTo()),
		FullServicePeriod: closedPeriodOptionFromDB(entity.GetOverrideFullServicePeriodFrom(), entity.GetOverrideFullServicePeriodTo()),
		BillingPeriod:     closedPeriodOptionFromDB(entity.GetOverrideBillingPeriodFrom(), entity.GetOverrideBillingPeriodTo()),
	}

	return &override
}

func optionStringPtrFromDB(value *string) mo.Option[*string] {
	if value == nil {
		return mo.None[*string]()
	}

	if *value == "" {
		return mo.Some((*string)(nil))
	}

	return mo.Some(value)
}

func stringPtrToDB(value mo.Option[*string]) *string {
	if !value.IsPresent() {
		return nil
	}

	if value.OrEmpty() == nil {
		return lo.ToPtr("")
	}

	return value.OrEmpty()
}

func taxBehaviorOverrideFromOption(value mo.Option[*productcatalog.TaxBehavior]) TaxBehaviorOverride {
	taxBehavior := value.OrEmpty()
	if taxBehavior == nil {
		return TaxBehaviorOverrideClear
	}

	return TaxBehaviorOverride(*taxBehavior)
}

func closedPeriodOptionFromDB(from, to *time.Time) *timeutil.ClosedPeriod {
	if from == nil || to == nil {
		return nil
	}

	return &timeutil.ClosedPeriod{
		From: from.UTC(),
		To:   to.UTC(),
	}
}

// Nillable pointer GoTypes generate SetOrClear methods that accept **T, where nil
// clears the DB column and *T stores the pointed-to value.
func fromOptionalPtrToSetOrClear[T any](value *T) **T {
	if value == nil {
		return nil
	}

	return lo.ToPtr(value)
}

func clearOnBaseUpdater[T BaseUpdater[T]](updater T) T {
	return updater.
		ClearOverrideKind().
		ClearOverrideName().
		ClearOverrideDescription().
		ClearOverrideMetadata().
		ClearOverrideTaxBehavior().
		ClearOverrideTaxCodeID().
		ClearOverrideServicePeriodFrom().
		ClearOverrideServicePeriodTo().
		ClearOverrideFullServicePeriodFrom().
		ClearOverrideFullServicePeriodTo().
		ClearOverrideBillingPeriodFrom().
		ClearOverrideBillingPeriodTo()
}
