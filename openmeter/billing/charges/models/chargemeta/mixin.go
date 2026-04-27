package chargemeta

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type Mixin = entutils.RecursiveMixin[metaMixin]

type metaMixin struct {
	mixin.Schema
}

func (metaMixin) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entutils.AnnotationsMixin{},
		entutils.ResourceMixin{},
	}
}

func (metaMixin) Fields() []ent.Field {
	return []ent.Field{
		field.String("customer_id").
			NotEmpty().
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),

		field.Time("service_period_from"),
		field.Time("service_period_to"),
		field.Time("billing_period_from"),
		field.Time("billing_period_to"),
		field.Time("full_service_period_from"),
		field.Time("full_service_period_to"),

		field.Enum("status").
			GoType(meta.ChargeStatus("")),

		field.String("unique_reference_id").
			Immutable().
			Optional().
			Nillable(),

		field.String("currency").
			GoType(currencyx.Code("")).
			NotEmpty().
			Immutable().
			SchemaType(map[string]string{
				dialect.Postgres: "varchar(3)",
			}),

		field.Enum("managed_by").
			GoType(billing.InvoiceLineManagedBy("")),

		// Subscriptions metadata
		field.String("subscription_id").
			Optional().
			Nillable().
			Immutable(),

		field.String("subscription_phase_id").
			Optional().
			Nillable().
			Immutable(),

		field.String("subscription_item_id").
			Optional().
			Nillable().
			Immutable(),

		field.Time("advance_after").
			Optional().
			Nillable(),
		field.String("tax_code_id").
			Optional().
			Nillable().
			SchemaType(map[string]string{
				dialect.Postgres: "char(26)",
			}),
		field.Enum("tax_behavior").
			GoType(productcatalog.TaxBehavior("")).
			Optional().
			Nillable(),
	}
}

func (metaMixin) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("namespace", "customer_id", "unique_reference_id").
			Annotations(
				entsql.IndexWhere("unique_reference_id IS NOT NULL AND deleted_at IS NULL"),
			).
			Unique(),
		index.Fields("tax_code_id").
			Annotations(entsql.IndexWhere("tax_code_id IS NOT NULL AND deleted_at IS NULL")),
	}
}

type CreateInput struct {
	Namespace string

	Intent meta.Intent

	Status       meta.ChargeStatus
	AdvanceAfter *time.Time
}

type Creator[T any] interface {
	entutils.NamespaceMixinCreator[T]
	entutils.AnnotationsMixinSetter[T]
	entutils.TimeMixinCreator[T]

	SetCustomerID(customerID string) T
	SetCurrency(currency currencyx.Code) T
	SetNillableUniqueReferenceID(uniqueReferenceID *string) T
	SetNillableSubscriptionID(subscriptionID *string) T
	SetNillableSubscriptionPhaseID(subscriptionPhaseID *string) T
	SetNillableSubscriptionItemID(subscriptionItemID *string) T

	// Mutable fields
	SetName(name string) T
	SetNillableDescription(description *string) T
	SetMetadata(metadata map[string]string) T
	SetAnnotations(annotations models.Annotations) T
	SetServicePeriodFrom(servicePeriodFrom time.Time) T
	SetServicePeriodTo(servicePeriodTo time.Time) T
	SetBillingPeriodFrom(billingPeriodFrom time.Time) T
	SetBillingPeriodTo(billingPeriodTo time.Time) T
	SetFullServicePeriodFrom(fullServicePeriodFrom time.Time) T
	SetFullServicePeriodTo(fullServicePeriodTo time.Time) T
	SetStatus(status meta.ChargeStatus) T
	SetNillableAdvanceAfter(advanceAfter *time.Time) T
	SetManagedBy(managedBy billing.InvoiceLineManagedBy) T
	SetNillableTaxCodeID(taxCodeID *string) T
	SetNillableTaxBehavior(taxBehavior *productcatalog.TaxBehavior) T
}

type Updater[T any] interface {
	SetName(name string) T
	SetOrClearDescription(description *string) T
	SetMetadata(metadata map[string]string) T
	SetAnnotations(annotations models.Annotations) T
	SetServicePeriodFrom(servicePeriodFrom time.Time) T
	SetServicePeriodTo(servicePeriodTo time.Time) T
	SetBillingPeriodFrom(billingPeriodFrom time.Time) T
	SetBillingPeriodTo(billingPeriodTo time.Time) T
	SetFullServicePeriodFrom(fullServicePeriodFrom time.Time) T
	SetFullServicePeriodTo(fullServicePeriodTo time.Time) T
	SetStatus(status meta.ChargeStatus) T
	SetOrClearAdvanceAfter(advanceAfter *time.Time) T
	SetOrClearDeletedAt(deletedAt *time.Time) T
	SetManagedBy(managedBy billing.InvoiceLineManagedBy) T
	SetOrClearTaxCodeID(taxCodeID *string) T
	SetOrClearTaxBehavior(taxBehavior *productcatalog.TaxBehavior) T
}

func Create[T Creator[T]](creator Creator[T], in CreateInput) (T, error) {
	in.Intent = in.Intent.Normalized()
	in.AdvanceAfter = meta.NormalizeOptionalTimestamp(in.AdvanceAfter)

	if err := in.Intent.Validate(); err != nil {
		var empty T
		return empty, err
	}

	var subscriptionID *string
	if in.Intent.Subscription != nil {
		subscriptionID = &in.Intent.Subscription.SubscriptionID
	}
	var subscriptionPhaseID *string
	if in.Intent.Subscription != nil {
		subscriptionPhaseID = &in.Intent.Subscription.PhaseID
	}
	var subscriptionItemID *string
	if in.Intent.Subscription != nil {
		subscriptionItemID = &in.Intent.Subscription.ItemID
	}

	var taxCodeID *string
	var taxBehavior *productcatalog.TaxBehavior
	if in.Intent.TaxConfig != nil {
		taxCodeID = in.Intent.TaxConfig.TaxCodeID
		taxBehavior = in.Intent.TaxConfig.Behavior
	}

	return creator.
		SetNamespace(in.Namespace).
		SetName(in.Intent.Name).
		SetNillableDescription(in.Intent.Description).
		SetMetadata(in.Intent.Metadata).
		SetAnnotations(in.Intent.Annotations).
		SetCustomerID(in.Intent.CustomerID).
		SetServicePeriodFrom(in.Intent.ServicePeriod.From.UTC()).
		SetServicePeriodTo(in.Intent.ServicePeriod.To.UTC()).
		SetBillingPeriodFrom(in.Intent.BillingPeriod.From.UTC()).
		SetBillingPeriodTo(in.Intent.BillingPeriod.To.UTC()).
		SetFullServicePeriodFrom(in.Intent.FullServicePeriod.From.UTC()).
		SetFullServicePeriodTo(in.Intent.FullServicePeriod.To.UTC()).
		SetStatus(in.Status).
		SetCurrency(in.Intent.Currency).
		SetManagedBy(in.Intent.ManagedBy).
		SetNillableUniqueReferenceID(in.Intent.UniqueReferenceID).
		SetNillableAdvanceAfter(convert.SafeToUTC(in.AdvanceAfter)).
		SetNillableSubscriptionID(subscriptionID).
		SetNillableSubscriptionPhaseID(subscriptionPhaseID).
		SetNillableSubscriptionItemID(subscriptionItemID).
		SetNillableTaxCodeID(taxCodeID).
		SetNillableTaxBehavior(taxBehavior), nil
}

type UpdateInput struct {
	meta.ManagedResource
	Intent meta.Intent

	Status       meta.ChargeStatus
	AdvanceAfter *time.Time
}

func Update[T Updater[T]](updater Updater[T], in UpdateInput) (T, error) {
	in.Intent = in.Intent.Normalized()
	in.AdvanceAfter = meta.NormalizeOptionalTimestamp(in.AdvanceAfter)

	var taxCodeID *string
	var taxBehavior *productcatalog.TaxBehavior
	if in.Intent.TaxConfig != nil {
		taxCodeID = in.Intent.TaxConfig.TaxCodeID
		taxBehavior = in.Intent.TaxConfig.Behavior
	}

	return updater.
		SetName(in.Intent.Name).
		SetOrClearDeletedAt(convert.TimePtrIn(in.DeletedAt, time.UTC)).
		SetOrClearDescription(in.Intent.Description).
		SetMetadata(in.Intent.Metadata).
		SetAnnotations(in.Intent.Annotations).
		SetServicePeriodFrom(in.Intent.ServicePeriod.From.UTC()).
		SetServicePeriodTo(in.Intent.ServicePeriod.To.UTC()).
		SetBillingPeriodFrom(in.Intent.BillingPeriod.From.UTC()).
		SetBillingPeriodTo(in.Intent.BillingPeriod.To.UTC()).
		SetFullServicePeriodFrom(in.Intent.FullServicePeriod.From.UTC()).
		SetFullServicePeriodTo(in.Intent.FullServicePeriod.To.UTC()).
		SetStatus(in.Status).
		SetOrClearAdvanceAfter(in.AdvanceAfter).
		SetManagedBy(in.Intent.ManagedBy).
		SetOrClearTaxCodeID(taxCodeID).
		SetOrClearTaxBehavior(taxBehavior), nil
}

type Getter[T any] interface {
	entutils.TimeMixinGetter
	entutils.NamespaceMixinGetter
	entutils.IDMixinGetter
	entutils.AnnotationsMixinGetter

	GetStatus() meta.ChargeStatus
	GetName() string
	GetDescription() *string
	GetMetadata() map[string]string
	GetAnnotations() models.Annotations
	GetManagedBy() billing.InvoiceLineManagedBy
	GetCustomerID() string
	GetCurrency() currencyx.Code
	GetServicePeriodFrom() time.Time
	GetServicePeriodTo() time.Time
	GetAdvanceAfter() *time.Time
	GetFullServicePeriodFrom() time.Time
	GetFullServicePeriodTo() time.Time
	GetBillingPeriodFrom() time.Time
	GetBillingPeriodTo() time.Time
	GetUniqueReferenceID() *string
	GetSubscriptionID() *string
	GetSubscriptionPhaseID() *string
	GetSubscriptionItemID() *string
	GetTaxCodeID() *string
	GetTaxBehavior() *productcatalog.TaxBehavior
}

func MapFromDB[T Getter[T]](entity T) meta.Charge {
	var subscriptionReference *meta.SubscriptionReference
	if entity.GetSubscriptionID() != nil && entity.GetSubscriptionPhaseID() != nil && entity.GetSubscriptionItemID() != nil {
		subscriptionReference = &meta.SubscriptionReference{
			SubscriptionID: *entity.GetSubscriptionID(),
			PhaseID:        *entity.GetSubscriptionPhaseID(),
			ItemID:         *entity.GetSubscriptionItemID(),
		}
	}

	// Charge tables persist only TaxCodeID (FK) and Behavior; TaxConfig.Stripe is resolved at
	// invoice snapshot time from the referenced TaxCode and is not stored on the charge row.
	var taxConfig *productcatalog.TaxConfig
	if entity.GetTaxCodeID() != nil || entity.GetTaxBehavior() != nil {
		taxConfig = &productcatalog.TaxConfig{
			TaxCodeID: entity.GetTaxCodeID(),
			Behavior:  entity.GetTaxBehavior(),
		}
	}

	return meta.Charge{
		ManagedResource: meta.ManagedResource{
			NamespacedModel: models.NamespacedModel{
				Namespace: entity.GetNamespace(),
			},
			ManagedModel: entutils.MapTimeMixinFromDB(entity),
			ID:           entity.GetID(),
		},
		Intent: meta.Intent{
			Name:        entity.GetName(),
			Description: entity.GetDescription(),
			Metadata:    entity.GetMetadata(),
			Annotations: entity.GetAnnotations(),
			ManagedBy:   entity.GetManagedBy(),
			CustomerID:  entity.GetCustomerID(),
			Currency:    entity.GetCurrency(),
			ServicePeriod: timeutil.ClosedPeriod{
				From: entity.GetServicePeriodFrom().UTC(),
				To:   entity.GetServicePeriodTo().UTC(),
			},
			FullServicePeriod: timeutil.ClosedPeriod{
				From: entity.GetFullServicePeriodFrom().UTC(),
				To:   entity.GetFullServicePeriodTo().UTC(),
			},
			BillingPeriod: timeutil.ClosedPeriod{
				From: entity.GetBillingPeriodFrom().UTC(),
				To:   entity.GetBillingPeriodTo().UTC(),
			},
			UniqueReferenceID: entity.GetUniqueReferenceID(),
			Subscription:      subscriptionReference,
			TaxConfig:         taxConfig,
		},
		Status:       entity.GetStatus(),
		AdvanceAfter: entity.GetAdvanceAfter(),
	}
}
