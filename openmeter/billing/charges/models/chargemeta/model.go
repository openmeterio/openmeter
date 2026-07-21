package chargemeta

import (
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	currenciesadapter "github.com/openmeterio/openmeter/openmeter/currencies/adapter"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/entutils/entedge"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type CreateInput struct {
	Namespace string

	Intent              meta.Intent
	IntentMutableFields meta.IntentMutableFields

	Status       meta.ChargeStatus
	AdvanceAfter *time.Time
}

func (i CreateInput) Validate() error {
	var errs []error

	if err := i.Intent.Validate(); err != nil {
		errs = append(errs, err)
	}

	if err := i.IntentMutableFields.Validate(); err != nil {
		errs = append(errs, err)
	}

	if i.Namespace == "" {
		errs = append(errs, fmt.Errorf("namespace is required"))
	}

	if i.Status == "" {
		errs = append(errs, fmt.Errorf("status is required"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type Creator[T any] interface {
	entutils.NamespaceMixinCreator[T]
	entutils.AnnotationsMixinSetter[T]
	entutils.TimeMixinCreator[T]

	SetCustomerID(customerID string) T
	SetNillableFiatCurrencyCode(currency *currencyx.Code) T
	SetNillableCustomCurrencyID(customCurrencyID *string) T
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
	SetTaxCodeID(taxCodeID string) T
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
}

func Create[T Creator[T]](creator Creator[T], in CreateInput) (T, error) {
	in.IntentMutableFields = in.IntentMutableFields.Normalized()
	in.AdvanceAfter = meta.NormalizeOptionalTimestamp(in.AdvanceAfter)

	if err := in.Validate(); err != nil {
		return lo.Empty[T](), err
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

	switch in.Intent.Currency.Type() {
	case currencyx.CurrencyTypeFiat:
		def := in.Intent.Currency.Definition()

		if def == nil {
			return lo.Empty[T](), fmt.Errorf("resolved currency definition is required")
		}

		creator = creator.SetNillableFiatCurrencyCode(lo.ToPtr(currencyx.Code(def.ISOCode)))

	case currencyx.CurrencyTypeCustom:
		if in.Intent.Currency.ID == "" {
			return lo.Empty[T](), fmt.Errorf("resolved currency ID is required")
		}

		creator = creator.SetNillableCustomCurrencyID(lo.ToPtr(in.Intent.Currency.ID))
	default:
		return lo.Empty[T](), fmt.Errorf("unsupported currency type: %s", in.Intent.Currency.Type())
	}

	return creator.
		SetNamespace(in.Namespace).
		SetName(in.IntentMutableFields.Name).
		SetNillableDescription(in.IntentMutableFields.Description).
		SetMetadata(in.IntentMutableFields.Metadata).
		SetAnnotations(in.Intent.Annotations).
		SetCustomerID(in.Intent.CustomerID).
		SetServicePeriodFrom(in.IntentMutableFields.ServicePeriod.From.UTC()).
		SetServicePeriodTo(in.IntentMutableFields.ServicePeriod.To.UTC()).
		SetBillingPeriodFrom(in.IntentMutableFields.BillingPeriod.From.UTC()).
		SetBillingPeriodTo(in.IntentMutableFields.BillingPeriod.To.UTC()).
		SetFullServicePeriodFrom(in.IntentMutableFields.FullServicePeriod.From.UTC()).
		SetFullServicePeriodTo(in.IntentMutableFields.FullServicePeriod.To.UTC()).
		SetStatus(in.Status).
		SetManagedBy(in.Intent.ManagedBy).
		SetNillableUniqueReferenceID(in.Intent.UniqueReferenceID).
		SetNillableAdvanceAfter(convert.SafeToUTC(in.AdvanceAfter)).
		SetNillableSubscriptionID(subscriptionID).
		SetNillableSubscriptionPhaseID(subscriptionPhaseID).
		SetNillableSubscriptionItemID(subscriptionItemID).
		SetTaxCodeID(in.Intent.TaxConfig.TaxCodeID).
		SetNillableTaxBehavior(in.Intent.TaxConfig.Behavior), nil
}

type UpdateInput struct {
	meta.ManagedResource
	Intent              meta.Intent
	IntentMutableFields meta.IntentMutableFields

	Status       meta.ChargeStatus
	AdvanceAfter *time.Time
}

func Update[T Updater[T]](updater Updater[T], in UpdateInput) (T, error) {
	in.IntentMutableFields = in.IntentMutableFields.Normalized()
	in.AdvanceAfter = meta.NormalizeOptionalTimestamp(in.AdvanceAfter)

	if err := in.IntentMutableFields.Validate(); err != nil {
		var empty T
		return empty, err
	}

	if err := in.Intent.Validate(); err != nil {
		var empty T
		return empty, err
	}

	return updater.
		SetName(in.IntentMutableFields.Name).
		SetOrClearDescription(in.IntentMutableFields.Description).
		SetMetadata(in.IntentMutableFields.Metadata).
		SetAnnotations(in.Intent.Annotations).
		SetServicePeriodFrom(in.IntentMutableFields.ServicePeriod.From.UTC()).
		SetServicePeriodTo(in.IntentMutableFields.ServicePeriod.To.UTC()).
		SetBillingPeriodFrom(in.IntentMutableFields.BillingPeriod.From.UTC()).
		SetBillingPeriodTo(in.IntentMutableFields.BillingPeriod.To.UTC()).
		SetFullServicePeriodFrom(in.IntentMutableFields.FullServicePeriod.From.UTC()).
		SetFullServicePeriodTo(in.IntentMutableFields.FullServicePeriod.To.UTC()).
		SetStatus(in.Status).
		SetOrClearAdvanceAfter(in.AdvanceAfter), nil
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
	GetTaxCodeID() string
	GetTaxBehavior() *productcatalog.TaxBehavior
	GetFiatCurrencyCode() *currencyx.Code
	GetCustomCurrencyID() *string
}

type EdgeGetter interface {
	CustomCurrencyOrErr() (*entdb.CustomCurrency, error)
}

func FromDB[T Getter[T]](entity T, edges EdgeGetter) (meta.Charge, error) {
	var dbCustomCurrency *entdb.CustomCurrency
	if entity.GetCustomCurrencyID() != nil {
		var err error
		dbCustomCurrency, err = entedge.OrNilIfNotFound(edges.CustomCurrencyOrErr())
		if err != nil {
			return meta.Charge{}, fmt.Errorf("failed to get custom currency: %w", err)
		}
	}

	resolvedCurrency, err := currenciesadapter.FromDBCustomCurrencyOrFiatCurrency(currenciesadapter.CustomCurrencyOrFiatCurrency{
		CustomCurrency: dbCustomCurrency,
		FiatCurrency:   entity.GetFiatCurrencyCode(),
	})
	if err != nil {
		return meta.Charge{}, fmt.Errorf("failed to resolve currency: %w", err)
	}

	return FromDBWithCurrency(entity, resolvedCurrency)
}

func FromDBWithCurrency[T Getter[T]](entity T, currency currencies.Currency) (meta.Charge, error) {
	if err := currency.Validate(); err != nil {
		return meta.Charge{}, fmt.Errorf("currency: %w", err)
	}

	var subscriptionReference *meta.SubscriptionReference
	if entity.GetSubscriptionID() != nil && entity.GetSubscriptionPhaseID() != nil && entity.GetSubscriptionItemID() != nil {
		subscriptionReference = &meta.SubscriptionReference{
			SubscriptionID: *entity.GetSubscriptionID(),
			PhaseID:        *entity.GetSubscriptionPhaseID(),
			ItemID:         *entity.GetSubscriptionItemID(),
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
			ManagedBy:   entity.GetManagedBy(),
			CustomerID:  entity.GetCustomerID(),
			Annotations: entity.GetAnnotations(),
			Currency:    currency,
			TaxConfig: productcatalog.TaxCodeConfig{
				TaxCodeID: entity.GetTaxCodeID(),
				Behavior:  entity.GetTaxBehavior(),
			},
			UniqueReferenceID: entity.GetUniqueReferenceID(),
			Subscription:      subscriptionReference,
		},
		IntentMutableFields: meta.IntentMutableFields{
			Name:        entity.GetName(),
			Description: entity.GetDescription(),
			Metadata:    entity.GetMetadata(),
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
		},
		Status:       entity.GetStatus(),
		AdvanceAfter: entity.GetAdvanceAfter(),
	}, nil
}
